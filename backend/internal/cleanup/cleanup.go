package cleanup

import (
	"fmt"
	"log"
	"real-estate-portal/internal/models"
	"time"

	"gorm.io/gorm"
)

// Service handles physical deletion of old removed properties
type Service struct {
	db *gorm.DB
}

// NewService creates a new cleanup service
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// CleanupConfig holds configuration for cleanup operations
type CleanupConfig struct {
	RetentionDays      int  // Days to keep removed properties before physical deletion (default: 90)
	MaxDeletionCount   int  // Maximum number of properties to delete in one run (safety limit)
	DryRun             bool // If true, only log what would be deleted without actually deleting
	DeleteFromSearch   bool // If true, also delete from Meilisearch
}

// DefaultCleanupConfig returns default configuration
func DefaultCleanupConfig() CleanupConfig {
	return CleanupConfig{
		RetentionDays:    90,
		MaxDeletionCount: 10000,
		DryRun:           false,
		DeleteFromSearch: true,
	}
}

// CleanupResult holds the result of a cleanup operation
type CleanupResult struct {
	TargetCount       int       `json:"target_count"`        // Number of properties eligible for deletion
	DeletedCount      int       `json:"deleted_count"`       // Number of properties actually deleted
	SkippedCount      int       `json:"skipped_count"`       // Number of properties skipped
	ErrorCount        int       `json:"error_count"`         // Number of errors encountered
	DryRun            bool      `json:"dry_run"`             // Whether this was a dry run
	ExecutedAt        time.Time `json:"executed_at"`         // When the cleanup was executed
	DeletedProperties []string  `json:"deleted_properties"`  // IDs of deleted properties
	Errors            []string  `json:"errors,omitempty"`    // Error messages
}

// FindExpiredProperties finds properties that are eligible for physical deletion
// Properties must be:
// 1. Status = 'removed'
// 2. removed_at is older than retentionDays
func (s *Service) FindExpiredProperties(retentionDays int) ([]models.Property, error) {
	var properties []models.Property

	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)

	err := s.db.Where("status = ? AND removed_at < ?",
		models.PropertyStatusRemoved,
		cutoffDate,
	).Find(&properties).Error

	if err != nil {
		return nil, fmt.Errorf("failed to find expired properties: %w", err)
	}

	log.Printf("Found %d properties expired before %s", len(properties), cutoffDate.Format("2006-01-02"))
	return properties, nil
}

// PhysicallyDelete performs physical deletion of properties
func (s *Service) PhysicallyDelete(config CleanupConfig) (*CleanupResult, error) {
	result := &CleanupResult{
		DryRun:     config.DryRun,
		ExecutedAt: time.Now(),
	}

	// Find expired properties
	expiredProperties, err := s.FindExpiredProperties(config.RetentionDays)
	if err != nil {
		return nil, err
	}

	result.TargetCount = len(expiredProperties)

	if result.TargetCount == 0 {
		log.Println("No expired properties found for deletion")
		return result, nil
	}

	// Safety check: abort if too many properties would be deleted
	if result.TargetCount > config.MaxDeletionCount {
		return nil, fmt.Errorf("safety check failed: %d properties exceed max deletion limit of %d",
			result.TargetCount, config.MaxDeletionCount)
	}

	log.Printf("Starting cleanup: %d properties to delete (retention: %d days, dry-run: %v)",
		result.TargetCount, config.RetentionDays, config.DryRun)

	// Process each property
	for _, prop := range expiredProperties {
		if config.DryRun {
			// Dry run: just log what would be deleted
			log.Printf("[DRY-RUN] Would delete property %s (Title: %s, RemovedAt: %s)",
				prop.ID, prop.Title, prop.RemovedAt.Format("2006-01-02"))
			result.DeletedProperties = append(result.DeletedProperties, prop.ID)
			result.DeletedCount++
			continue
		}

		// Begin transaction for atomic operation
		tx := s.db.Begin()

		// 1. Create delete log entry
		deleteLog := models.DeleteLog{
			PropertyID: prop.ID,
			Title:      prop.Title,
			DetailURL:  prop.DetailURL,
			RemovedAt:  *prop.RemovedAt,
			Reason:     models.DeleteReasonExpired,
		}

		if err := tx.Create(&deleteLog).Error; err != nil {
			tx.Rollback()
			errMsg := fmt.Sprintf("Failed to create delete log for property %s: %v", prop.ID, err)
			log.Printf("ERROR: %s", errMsg)
			result.Errors = append(result.Errors, errMsg)
			result.ErrorCount++
			continue
		}

		// 2. Delete associated snapshots (optional - keep for history)
		// Uncomment if you want to delete snapshots:
		// if err := tx.Where("property_id = ?", prop.ID).Delete(&models.PropertySnapshot{}).Error; err != nil {
		// 	tx.Rollback()
		// 	errMsg := fmt.Sprintf("Failed to delete snapshots for property %s: %v", prop.ID, err)
		// 	log.Printf("ERROR: %s", errMsg)
		// 	result.Errors = append(result.Errors, errMsg)
		// 	result.ErrorCount++
		// 	continue
		// }

		// 3. Delete the property record
		if err := tx.Delete(&prop).Error; err != nil {
			tx.Rollback()
			errMsg := fmt.Sprintf("Failed to delete property %s: %v", prop.ID, err)
			log.Printf("ERROR: %s", errMsg)
			result.Errors = append(result.Errors, errMsg)
			result.ErrorCount++
			continue
		}

		// Commit transaction
		if err := tx.Commit().Error; err != nil {
			errMsg := fmt.Sprintf("Failed to commit deletion for property %s: %v", prop.ID, err)
			log.Printf("ERROR: %s", errMsg)
			result.Errors = append(result.Errors, errMsg)
			result.ErrorCount++
			continue
		}

		log.Printf("Physically deleted property %s (Title: %s)", prop.ID, prop.Title)
		result.DeletedProperties = append(result.DeletedProperties, prop.ID)
		result.DeletedCount++
	}

	log.Printf("Cleanup completed: %d/%d deleted, %d errors (dry-run: %v)",
		result.DeletedCount, result.TargetCount, result.ErrorCount, config.DryRun)

	return result, nil
}

// GetDeleteStats returns statistics about deleted properties
func (s *Service) GetDeleteStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total delete logs
	var totalDeleted int64
	if err := s.db.Model(&models.DeleteLog{}).Count(&totalDeleted).Error; err != nil {
		return nil, err
	}
	stats["total_deleted"] = totalDeleted

	// Delete logs by reason
	var reasonCounts []struct {
		Reason string
		Count  int64
	}
	if err := s.db.Model(&models.DeleteLog{}).
		Select("reason, count(*) as count").
		Group("reason").
		Scan(&reasonCounts).Error; err != nil {
		return nil, err
	}

	reasonMap := make(map[string]int64)
	for _, rc := range reasonCounts {
		reasonMap[rc.Reason] = rc.Count
	}
	stats["by_reason"] = reasonMap

	// Recent deletions (last 30 days)
	var recentDeleted int64
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	if err := s.db.Model(&models.DeleteLog{}).
		Where("deleted_at >= ?", thirtyDaysAgo).
		Count(&recentDeleted).Error; err != nil {
		return nil, err
	}
	stats["deleted_last_30_days"] = recentDeleted

	// Current removed count (pending deletion)
	var currentRemoved int64
	if err := s.db.Model(&models.Property{}).
		Where("status = ?", models.PropertyStatusRemoved).
		Count(&currentRemoved).Error; err != nil {
		return nil, err
	}
	stats["currently_removed"] = currentRemoved

	// Expired count (ready for deletion)
	expiredProperties, err := s.FindExpiredProperties(90)
	if err != nil {
		return nil, err
	}
	stats["expired_ready_for_deletion"] = len(expiredProperties)

	return stats, nil
}

// GetRecentDeleteLogs returns recent delete log entries
func (s *Service) GetRecentDeleteLogs(limit int) ([]models.DeleteLog, error) {
	var logs []models.DeleteLog
	err := s.db.Order("deleted_at DESC").Limit(limit).Find(&logs).Error
	return logs, err
}
