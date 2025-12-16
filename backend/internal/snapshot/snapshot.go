package snapshot

import (
	"fmt"
	"log"
	"real-estate-portal/internal/models"
	"time"

	"gorm.io/gorm"
)

// Service handles property snapshot operations
type Service struct {
	db *gorm.DB
}

// NewService creates a new snapshot service
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// CreateSnapshot creates a snapshot of a property
func (s *Service) CreateSnapshot(property *models.Property) error {
	snapshot := &models.PropertySnapshot{
		PropertyID:  property.ID,
		SnapshotAt:  time.Now().Truncate(24 * time.Hour), // Truncate to date only
		Rent:        property.Rent,
		FloorPlan:   property.FloorPlan,
		Area:        property.Area,
		WalkTime:    property.WalkTime,
		Station:     property.Station,
		Address:     property.Address,
		BuildingAge: property.BuildingAge,
		Floor:       property.Floor,
		ImageURL:    property.ImageURL,
		Status:      string(property.Status),
		HasChanged:  false,
	}

	// Check if snapshot already exists for today
	var existing models.PropertySnapshot
	result := s.db.Where("property_id = ? AND snapshot_at = ?", property.ID, snapshot.SnapshotAt).First(&existing)

	if result.Error == gorm.ErrRecordNotFound {
		// Create new snapshot
		return s.db.Create(snapshot).Error
	} else if result.Error != nil {
		return result.Error
	}

	// Update existing snapshot
	snapshot.ID = existing.ID
	return s.db.Save(snapshot).Error
}

// DetectChanges compares current property state with the most recent snapshot
func (s *Service) DetectChanges(property *models.Property) ([]models.PropertyChange, error) {
	// Get the most recent snapshot (not today's)
	var lastSnapshot models.PropertySnapshot
	today := time.Now().Truncate(24 * time.Hour)

	result := s.db.Where("property_id = ? AND snapshot_at < ?", property.ID, today).
		Order("snapshot_at DESC").
		First(&lastSnapshot)

	if result.Error == gorm.ErrRecordNotFound {
		// No previous snapshot, this is a new property
		return []models.PropertyChange{{
			PropertyID: property.ID,
			ChangeType: models.ChangeTypeNew,
			NewValue:   "New property detected",
			DetectedAt: time.Now(),
		}}, nil
	} else if result.Error != nil {
		return nil, result.Error
	}

	// Compare and detect changes
	changes := []models.PropertyChange{}

	// Rent change
	if !intPtrEqual(property.Rent, lastSnapshot.Rent) {
		oldVal := "nil"
		newVal := "nil"
		var magnitude float64

		if lastSnapshot.Rent != nil {
			oldVal = fmt.Sprintf("%d", *lastSnapshot.Rent)
		}
		if property.Rent != nil {
			newVal = fmt.Sprintf("%d", *property.Rent)
		}

		if lastSnapshot.Rent != nil && property.Rent != nil {
			magnitude = float64(*property.Rent - *lastSnapshot.Rent)
		}

		changes = append(changes, models.PropertyChange{
			PropertyID:      property.ID,
			ChangeType:      models.ChangeTypeRent,
			OldValue:        oldVal,
			NewValue:        newVal,
			ChangeMagnitude: &magnitude,
			DetectedAt:      time.Now(),
		})
	}

	// Status change
	if string(property.Status) != lastSnapshot.Status {
		changes = append(changes, models.PropertyChange{
			PropertyID: property.ID,
			ChangeType: models.ChangeTypeStatus,
			OldValue:   lastSnapshot.Status,
			NewValue:   string(property.Status),
			DetectedAt: time.Now(),
		})
	}

	// Floor plan change
	if property.FloorPlan != lastSnapshot.FloorPlan {
		changes = append(changes, models.PropertyChange{
			PropertyID: property.ID,
			ChangeType: models.ChangeTypeFloorPlan,
			OldValue:   lastSnapshot.FloorPlan,
			NewValue:   property.FloorPlan,
			DetectedAt: time.Now(),
		})
	}

	// Area change
	if !float64PtrEqual(property.Area, lastSnapshot.Area) {
		oldVal := "nil"
		newVal := "nil"

		if lastSnapshot.Area != nil {
			oldVal = fmt.Sprintf("%.2f", *lastSnapshot.Area)
		}
		if property.Area != nil {
			newVal = fmt.Sprintf("%.2f", *property.Area)
		}

		changes = append(changes, models.PropertyChange{
			PropertyID: property.ID,
			ChangeType: models.ChangeTypeArea,
			OldValue:   oldVal,
			NewValue:   newVal,
			DetectedAt: time.Now(),
		})
	}

	// Building age change
	if !intPtrEqual(property.BuildingAge, lastSnapshot.BuildingAge) {
		oldVal := "nil"
		newVal := "nil"

		if lastSnapshot.BuildingAge != nil {
			oldVal = fmt.Sprintf("%d", *lastSnapshot.BuildingAge)
		}
		if property.BuildingAge != nil {
			newVal = fmt.Sprintf("%d", *property.BuildingAge)
		}

		changes = append(changes, models.PropertyChange{
			PropertyID: property.ID,
			ChangeType: models.ChangeTypeBuildingAge,
			OldValue:   oldVal,
			NewValue:   newVal,
			DetectedAt: time.Now(),
		})
	}

	// Image change
	if property.ImageURL != lastSnapshot.ImageURL {
		changes = append(changes, models.PropertyChange{
			PropertyID: property.ID,
			ChangeType: models.ChangeTypeImage,
			OldValue:   lastSnapshot.ImageURL,
			NewValue:   property.ImageURL,
			DetectedAt: time.Now(),
		})
	}

	return changes, nil
}

// SaveChanges saves detected changes to the database
func (s *Service) SaveChanges(changes []models.PropertyChange, snapshotID uint) error {
	if len(changes) == 0 {
		return nil
	}

	// Set snapshot ID for all changes
	for i := range changes {
		changes[i].SnapshotID = snapshotID
	}

	return s.db.Create(&changes).Error
}

// CreateSnapshotWithChangeDetection creates a snapshot and detects changes
func (s *Service) CreateSnapshotWithChangeDetection(property *models.Property) error {
	// Detect changes first
	changes, err := s.DetectChanges(property)
	if err != nil {
		log.Printf("Warning: Failed to detect changes for property %s: %v", property.ID, err)
	}

	// Create snapshot
	snapshot := &models.PropertySnapshot{
		PropertyID:  property.ID,
		SnapshotAt:  time.Now().Truncate(24 * time.Hour),
		Rent:        property.Rent,
		FloorPlan:   property.FloorPlan,
		Area:        property.Area,
		WalkTime:    property.WalkTime,
		Station:     property.Station,
		Address:     property.Address,
		BuildingAge: property.BuildingAge,
		Floor:       property.Floor,
		ImageURL:    property.ImageURL,
		Status:      string(property.Status),
		HasChanged:  len(changes) > 0,
	}

	if len(changes) > 0 {
		changeNotes := []string{}
		for _, change := range changes {
			changeNotes = append(changeNotes, fmt.Sprintf("%s: %s -> %s", change.ChangeType, change.OldValue, change.NewValue))
		}
		snapshot.ChangeNote = fmt.Sprintf("%d changes detected", len(changes))
	}

	// Check if snapshot already exists for today
	var existing models.PropertySnapshot
	result := s.db.Where("property_id = ? AND snapshot_at = ?", property.ID, snapshot.SnapshotAt).First(&existing)

	if result.Error == gorm.ErrRecordNotFound {
		// Create new snapshot
		if err := s.db.Create(snapshot).Error; err != nil {
			return err
		}
	} else if result.Error != nil {
		return result.Error
	} else {
		// Update existing snapshot
		snapshot.ID = existing.ID
		if err := s.db.Save(snapshot).Error; err != nil {
			return err
		}
	}

	// Save changes
	if len(changes) > 0 {
		if err := s.SaveChanges(changes, snapshot.ID); err != nil {
			log.Printf("Warning: Failed to save changes: %v", err)
		} else {
			log.Printf("Detected %d changes for property %s", len(changes), property.ID)
		}
	}

	return nil
}

// GetPropertyHistory retrieves snapshot history for a property
func (s *Service) GetPropertyHistory(propertyID string, limit int) ([]models.PropertySnapshot, error) {
	var snapshots []models.PropertySnapshot
	query := s.db.Where("property_id = ?", propertyID).Order("snapshot_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&snapshots).Error; err != nil {
		return nil, err
	}

	return snapshots, nil
}

// GetRecentChanges retrieves recent property changes
func (s *Service) GetRecentChanges(limit int) ([]models.PropertyChange, error) {
	var changes []models.PropertyChange
	query := s.db.Order("detected_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&changes).Error; err != nil {
		return nil, err
	}

	return changes, nil
}

// Helper functions
func intPtrEqual(a, b *int) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func float64PtrEqual(a, b *float64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
