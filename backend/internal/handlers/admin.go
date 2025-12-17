package handlers

import (
	"log"
	"net/http"
	"real-estate-portal/internal/cleanup"
	"real-estate-portal/internal/models"
	"real-estate-portal/internal/scheduler"
	"real-estate-portal/internal/snapshot"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AdminHandler handles admin-related requests
type AdminHandler struct {
	db              *gorm.DB
	scheduler       *scheduler.Scheduler
	snapshotService *snapshot.Service
	cleanupService  *cleanup.Service
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(db *gorm.DB, sched *scheduler.Scheduler) *AdminHandler {
	return &AdminHandler{
		db:              db,
		scheduler:       sched,
		snapshotService: snapshot.NewService(db),
		cleanupService:  cleanup.NewService(db),
	}
}

// GetStats returns system statistics
func (h *AdminHandler) GetStats(c *gin.Context) {
	stats := make(map[string]interface{})

	// Property counts by status
	var activeCount, removedCount int64
	h.db.Model(&models.Property{}).Where("status = ?", models.PropertyStatusActive).Count(&activeCount)
	h.db.Model(&models.Property{}).Where("status = ?", models.PropertyStatusRemoved).Count(&removedCount)

	stats["properties"] = map[string]interface{}{
		"active":  activeCount,
		"removed": removedCount,
		"total":   activeCount + removedCount,
	}

	// Recent scraping activity (last 24 hours)
	last24h := time.Now().AddDate(0, 0, -1)
	var recentlyFetched int64
	h.db.Model(&models.Property{}).Where("fetched_at >= ?", last24h).Count(&recentlyFetched)
	stats["recent_activity"] = map[string]interface{}{
		"fetched_last_24h": recentlyFetched,
	}

	// Snapshot statistics
	var snapshotCount int64
	h.db.Model(&models.PropertySnapshot{}).Count(&snapshotCount)
	stats["snapshots"] = map[string]interface{}{
		"total": snapshotCount,
	}

	// Property changes (last 7 days)
	last7days := time.Now().AddDate(0, 0, -7)
	var recentChanges int64
	h.db.Model(&models.PropertyChange{}).Where("detected_at >= ?", last7days).Count(&recentChanges)
	stats["changes"] = map[string]interface{}{
		"last_7_days": recentChanges,
	}

	// Delete logs statistics
	deleteStats, err := h.cleanupService.GetDeleteStats()
	if err != nil {
		log.Printf("Failed to get delete stats: %v", err)
	} else {
		stats["deletions"] = deleteStats
	}

	c.JSON(http.StatusOK, stats)
}

// GetRecentActivity returns recent property activity
func (h *AdminHandler) GetRecentActivity(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	limit, _ := strconv.Atoi(limitStr)

	var properties []models.Property
	err := h.db.Order("fetched_at DESC").Limit(limit).Find(&properties).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"properties": properties,
		"count":      len(properties),
	})
}

// TriggerScraping manually triggers scraping
func (h *AdminHandler) TriggerScraping(c *gin.Context) {
	if h.scheduler == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Scheduler not available (MySQL/GORM required)",
		})
		return
	}

	log.Println("Admin: Manual scraping trigger requested")

	// Run in goroutine to avoid blocking
	go func() {
		if err := h.scheduler.RunNow(); err != nil {
			log.Printf("Admin: Manual scraping failed: %v", err)
		} else {
			log.Println("Admin: Manual scraping completed successfully")
		}
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Scraping job started",
		"status":  "running",
	})
}

// GetScrapingStatus returns current scraping status
func (h *AdminHandler) GetScrapingStatus(c *gin.Context) {
	// TODO: Implement actual status tracking
	// For now, return basic info
	c.JSON(http.StatusOK, gin.H{
		"status": "idle",
		"message": "Status tracking not yet implemented",
	})
}

// RunCleanup executes physical deletion of old removed properties
func (h *AdminHandler) RunCleanup(c *gin.Context) {
	var req struct {
		RetentionDays    int  `json:"retention_days"`     // Days to keep (default: 90)
		MaxDeletionCount int  `json:"max_deletion_count"` // Safety limit (default: 10000)
		DryRun           bool `json:"dry_run"`            // Dry run mode (default: true)
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set defaults
	config := cleanup.DefaultCleanupConfig()
	if req.RetentionDays > 0 {
		config.RetentionDays = req.RetentionDays
	}
	if req.MaxDeletionCount > 0 {
		config.MaxDeletionCount = req.MaxDeletionCount
	}
	config.DryRun = req.DryRun

	log.Printf("Admin: Running cleanup (retention: %d days, max: %d, dry-run: %v)",
		config.RetentionDays, config.MaxDeletionCount, config.DryRun)

	result, err := h.cleanupService.PhysicallyDelete(config)
	if err != nil {
		log.Printf("Admin: Cleanup failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Admin: Cleanup completed: %d/%d deleted (dry-run: %v)",
		result.DeletedCount, result.TargetCount, result.DryRun)

	c.JSON(http.StatusOK, result)
}

// GetDeleteLogs returns recent delete log entries
func (h *AdminHandler) GetDeleteLogs(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "100")
	limit, _ := strconv.Atoi(limitStr)

	logs, err := h.cleanupService.GetRecentDeleteLogs(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":  logs,
		"count": len(logs),
	})
}

// GetPropertyHistory returns snapshot history for a property
func (h *AdminHandler) GetPropertyHistory(c *gin.Context) {
	propertyID := c.Param("id")
	limitStr := c.DefaultQuery("limit", "30")
	limit, _ := strconv.Atoi(limitStr)

	snapshots, err := h.snapshotService.GetPropertyHistory(propertyID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"property_id": propertyID,
		"snapshots":   snapshots,
		"count":       len(snapshots),
	})
}

// GetRecentChanges returns recent property changes
func (h *AdminHandler) GetRecentChanges(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "100")
	limit, _ := strconv.Atoi(limitStr)

	changes, err := h.snapshotService.GetRecentChanges(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"changes": changes,
		"count":   len(changes),
	})
}

// GetAreaStats returns statistics by area
func (h *AdminHandler) GetAreaStats(c *gin.Context) {
	type AreaStat struct {
		Station string `json:"station"`
		Count   int64  `json:"count"`
	}

	var stats []AreaStat
	err := h.db.Model(&models.Property{}).
		Select("station, count(*) as count").
		Where("status = ? AND station IS NOT NULL AND station != ''", models.PropertyStatusActive).
		Group("station").
		Order("count DESC").
		Limit(20).
		Scan(&stats).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"area_stats": stats,
		"count":      len(stats),
	})
}

// GetPriceDistribution returns rent price distribution
func (h *AdminHandler) GetPriceDistribution(c *gin.Context) {
	type PriceRange struct {
		RangeLabel string `json:"range_label"`
		MinRent    int    `json:"min_rent"`
		MaxRent    int    `json:"max_rent"`
		Count      int64  `json:"count"`
	}

	// Define price ranges (in yen)
	ranges := []PriceRange{
		{RangeLabel: "〜5万円", MinRent: 0, MaxRent: 50000},
		{RangeLabel: "5〜8万円", MinRent: 50000, MaxRent: 80000},
		{RangeLabel: "8〜10万円", MinRent: 80000, MaxRent: 100000},
		{RangeLabel: "10〜15万円", MinRent: 100000, MaxRent: 150000},
		{RangeLabel: "15〜20万円", MinRent: 150000, MaxRent: 200000},
		{RangeLabel: "20万円〜", MinRent: 200000, MaxRent: 10000000},
	}

	for i := range ranges {
		var count int64
		h.db.Model(&models.Property{}).
			Where("status = ? AND rent >= ? AND rent < ?",
				models.PropertyStatusActive, ranges[i].MinRent, ranges[i].MaxRent).
			Count(&count)
		ranges[i].Count = count
	}

	c.JSON(http.StatusOK, gin.H{
		"price_distribution": ranges,
	})
}
