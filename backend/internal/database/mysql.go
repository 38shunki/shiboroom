package database

import (
	"crypto/md5"
	"fmt"
	"net/url"
	"real-estate-portal/internal/models"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type GormDB struct {
	db *gorm.DB
}

func NewGormDB(host, port, user, password, dbname string) (*GormDB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		user, password, host, port, dbname)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NowFunc: func() time.Time {
			return time.Now().Local()
		},
	})
	if err != nil {
		return nil, err
	}

	// Test connection
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	if err := sqlDB.Ping(); err != nil {
		return nil, err
	}

	return &GormDB{db: db}, nil
}

// NewGormDBFromDB creates a GormDB wrapper from an existing gorm.DB instance
func NewGormDBFromDB(db *gorm.DB) *GormDB {
	return &GormDB{db: db}
}

// DB returns the underlying gorm.DB instance
func (gdb *GormDB) DB() *gorm.DB {
	return gdb.db
}

func (gdb *GormDB) Close() error {
	sqlDB, err := gdb.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// GetDB returns the underlying gorm.DB instance
func (gdb *GormDB) GetDB() (*gorm.DB, error) {
	return gdb.db, nil
}

// InitSchema creates tables using GORM AutoMigrate
func (gdb *GormDB) InitSchema() error {
	// AutoMigrate will create tables if they don't exist
	return gdb.db.AutoMigrate(
		&models.Property{},
		&models.PropertySnapshot{},
		&models.PropertyChange{},
		&models.DeleteLog{},
		&models.DetailScrapeQueue{},
	)
}

// SaveProperty saves or updates a property (upsert by detail_url)
func (gdb *GormDB) SaveProperty(p *models.Property) error {
	// Generate ID from normalized URL if not set
	if p.ID == "" {
		normalizedURL := normalizeURL(p.DetailURL)
		p.ID = generateMD5(normalizedURL)
	}

	// Set FetchedAt to now if not set
	if p.FetchedAt.IsZero() {
		p.FetchedAt = time.Now()
	}

	// Set default status to active if not set
	if p.Status == "" {
		p.Status = models.PropertyStatusActive
	}

	// Upsert: try to create, on conflict (detail_url unique) update
	// First try to find existing property by detail_url
	var existing models.Property
	result := gdb.db.Where("detail_url = ?", p.DetailURL).First(&existing)

	if result.Error == gorm.ErrRecordNotFound {
		// Create new
		return gdb.db.Create(p).Error
	} else if result.Error != nil {
		return result.Error
	}

	// Update existing (keep original CreatedAt, Status, and RemovedAt)
	p.CreatedAt = existing.CreatedAt
	p.ID = existing.ID
	p.Status = existing.Status
	p.RemovedAt = existing.RemovedAt
	return gdb.db.Save(p).Error
}

// GetAllProperties retrieves all active properties
func (gdb *GormDB) GetAllProperties() ([]models.Property, error) {
	var properties []models.Property
	err := gdb.db.Order("created_at DESC").Find(&properties).Error
	return properties, err
}

// GetPropertiesWithSort retrieves all properties with custom sorting
func (gdb *GormDB) GetPropertiesWithSort(sortBy string) ([]models.Property, error) {
	var properties []models.Property

	// Map sort parameter to SQL ORDER BY clause (MySQL syntax)
	// Use CASE to put NULLs last for ASC, first for DESC
	var orderClause string
	switch sortBy {
	case "fetched_at", "fetched_at_desc":
		orderClause = "fetched_at DESC"
	case "fetched_at_asc":
		orderClause = "fetched_at ASC"
	case "rent_asc":
		orderClause = "CASE WHEN rent IS NULL THEN 1 ELSE 0 END, rent ASC"
	case "rent_desc":
		orderClause = "CASE WHEN rent IS NULL THEN 1 ELSE 0 END, rent DESC"
	case "area_desc":
		orderClause = "CASE WHEN area IS NULL THEN 1 ELSE 0 END, area DESC"
	case "walk_time_asc":
		orderClause = "CASE WHEN walk_time IS NULL THEN 1 ELSE 0 END, walk_time ASC"
	case "building_age_asc":
		orderClause = "CASE WHEN building_age IS NULL THEN 1 ELSE 0 END, building_age ASC"
	default:
		// Default to newest first (by fetched_at)
		orderClause = "fetched_at DESC"
	}

	err := gdb.db.Order(orderClause).Find(&properties).Error
	return properties, err
}

// GetPropertyByID retrieves a property by ID
func (gdb *GormDB) GetPropertyByID(id string) (*models.Property, error) {
	var property models.Property
	err := gdb.db.Where("id = ?", id).First(&property).Error
	if err != nil {
		return nil, err
	}
	return &property, nil
}

// savePropertyStations saves property stations within a transaction
// If stations is empty, does nothing (important: preserves existing data when HTML is missing/blocked)
func savePropertyStations(tx *gorm.DB, propertyID string, stations []models.PropertyStation) error {
	if len(stations) == 0 {
		// Important: Don't delete existing stations if extraction returns empty
		// (could be due to HTML missing, WAF block, or parsing error)
		return nil
	}

	// Delete existing stations for this property
	if err := tx.Where("property_id = ?", propertyID).Delete(&models.PropertyStation{}).Error; err != nil {
		return err
	}

	// Insert all new stations
	if len(stations) > 0 {
		if err := tx.Create(&stations).Error; err != nil {
			return err
		}
	}

	return nil
}

// SavePropertyWithStations saves a property and its stations in a transaction
func (gdb *GormDB) SavePropertyWithStations(p *models.Property, stations []models.PropertyStation) error {
	// Generate ID from normalized URL if not set
	if p.ID == "" {
		normalizedURL := normalizeURL(p.DetailURL)
		p.ID = generateMD5(normalizedURL)
	}

	// Set FetchedAt to now if not set
	if p.FetchedAt.IsZero() {
		p.FetchedAt = time.Now()
	}

	// Set default status to active if not set
	if p.Status == "" {
		p.Status = models.PropertyStatusActive
	}

	// Use transaction to save both property and stations
	return gdb.db.Transaction(func(tx *gorm.DB) error {
		// Upsert property: try to find existing
		var existing models.Property
		result := tx.Where("detail_url = ?", p.DetailURL).First(&existing)

		if result.Error == gorm.ErrRecordNotFound {
			// Create new property
			if err := tx.Create(p).Error; err != nil {
				return err
			}
		} else if result.Error != nil {
			return result.Error
		} else {
			// Update existing (keep original CreatedAt, Status, and RemovedAt)
			p.CreatedAt = existing.CreatedAt
			p.ID = existing.ID
			p.Status = existing.Status
			p.RemovedAt = existing.RemovedAt
			if err := tx.Save(p).Error; err != nil {
				return err
			}
		}

		// Save stations (only if extraction returned data)
		if err := savePropertyStations(tx, p.ID, stations); err != nil {
			return err
		}

		return nil
	})
}

// GetActiveProperties retrieves all active properties
func (gdb *GormDB) GetActiveProperties() ([]models.Property, error) {
	var properties []models.Property
	err := gdb.db.Where("status = ?", models.PropertyStatusActive).Order("created_at DESC").Find(&properties).Error
	return properties, err
}

// MarkPropertyAsRemoved marks a property as removed (logical deletion)
func (gdb *GormDB) MarkPropertyAsRemoved(id string) error {
	now := time.Now()
	return gdb.db.Model(&models.Property{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     models.PropertyStatusRemoved,
			"removed_at": &now,
		}).Error
}

// MarkPropertiesAsRemoved marks multiple properties as removed
func (gdb *GormDB) MarkPropertiesAsRemoved(ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	now := time.Now()
	return gdb.db.Model(&models.Property{}).
		Where("id IN ?", ids).
		Updates(map[string]interface{}{
			"status":     models.PropertyStatusRemoved,
			"removed_at": &now,
		}).Error
}

// DetectDifferences compares current active properties with newly scraped properties
// Returns: new IDs, removed IDs, updated properties
func (gdb *GormDB) DetectDifferences(scrapedProperties []models.Property) (newIDs []string, removedIDs []string, updatedProperties []models.Property, err error) {
	// Get all currently active properties
	activeProperties, err := gdb.GetActiveProperties()
	if err != nil {
		return nil, nil, nil, err
	}

	// Create maps for efficient lookup
	activeMap := make(map[string]*models.Property)
	for i := range activeProperties {
		activeMap[activeProperties[i].ID] = &activeProperties[i]
	}

	scrapedMap := make(map[string]*models.Property)
	for i := range scrapedProperties {
		scrapedMap[scrapedProperties[i].ID] = &scrapedProperties[i]
	}

	// Find new properties (in scraped but not in active)
	for id := range scrapedMap {
		if _, exists := activeMap[id]; !exists {
			newIDs = append(newIDs, id)
		}
	}

	// Find removed properties (in active but not in scraped)
	for id := range activeMap {
		if _, exists := scrapedMap[id]; !exists {
			removedIDs = append(removedIDs, id)
		}
	}

	// Find updated properties (in both, but content changed)
	for id, scrapedProp := range scrapedMap {
		if activeProp, exists := activeMap[id]; exists {
			// Check if key fields have changed
			if hasPropertyChanged(activeProp, scrapedProp) {
				updatedProperties = append(updatedProperties, *scrapedProp)
			}
		}
	}

	return newIDs, removedIDs, updatedProperties, nil
}

// hasPropertyChanged checks if property data has changed
func hasPropertyChanged(old, new *models.Property) bool {
	// Compare key fields that might change
	if old.Title != new.Title {
		return true
	}
	if old.Rent != new.Rent {
		return true
	}
	if old.ImageURL != new.ImageURL {
		return true
	}
	// Add more field comparisons as needed
	return false
}

// normalizeURL normalizes a URL for consistent ID generation
func normalizeURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	// Remove query parameters and fragment
	u.RawQuery = ""
	u.Fragment = ""

	// Ensure trailing slash consistency (remove it)
	u.Path = strings.TrimSuffix(u.Path, "/")

	// Force HTTPS
	u.Scheme = "https"

	return u.String()
}

// generateMD5 generates MD5 hash for a string
func generateMD5(text string) string {
	hash := md5.Sum([]byte(text))
	return fmt.Sprintf("%x", hash)
}
