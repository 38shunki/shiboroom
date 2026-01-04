package database

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
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
		&models.PropertyStation{},
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

// PropertyFilters holds filter parameters for property search
type PropertyFilters struct {
	// Station/Line filters
	Station  string // Partial match on station_name
	Line     string // Partial match on line_name
	MaxWalk  int    // Maximum walk time in minutes (0 = no filter)
	WalkMode string // "nearest" (use properties.walk_time) or "any" (use property_stations)

	// Range filters
	MinRent        *int     // Minimum rent (万円単位)
	MaxRent        *int     // Maximum rent (万円単位)
	MinArea        *float64 // Minimum area (㎡)
	MaxArea        *float64 // Maximum area (㎡)
	MinBuildingAge *int     // Minimum building age (years)
	MaxBuildingAge *int     // Maximum building age (years)
	MinFloor       *int     // Minimum floor
	MaxFloor       *int     // Maximum floor

	// Multi-select filters
	FloorPlans     []string // Floor plan types (1K, 1DK, etc.)
	BuildingTypes  []string // Building types (mansion, apartment, etc.)
	Facilities     []string // Required facilities

	// Exclude filters
	ExcludeIDs     []string // Property IDs to exclude
	ExcludeStatuses []string // Statuses to exclude (default: exclude "removed")

	// Sort & Pagination
	SortBy   string // Sort parameter
	Limit    int    // Number of records to return (default: 50, max: 20000)
	Offset   *int   // Number of records to skip (legacy, optional)
	Cursor   string // Cursor for keyset pagination (new method)
}

// PaginatedPropertiesResponse holds paginated property results
type PaginatedPropertiesResponse struct {
	Properties []models.Property `json:"properties"`
	Total      int64             `json:"total"`
	Limit      int               `json:"limit"`
	Offset     int               `json:"offset,omitempty"`     // Legacy field (optional)
	NextCursor string            `json:"next_cursor,omitempty"` // Cursor for next page
}

// CursorData holds the decoded cursor information
type CursorData struct {
	FetchedAt string `json:"t"`  // Timestamp in RFC3339 format
	ID        string `json:"id"` // Property ID
}

// DecodeCursor decodes a base64-encoded JSON cursor
func DecodeCursor(cursor string) (*CursorData, error) {
	if cursor == "" {
		return nil, nil
	}

	// Decode base64
	decoded, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor: base64 decode failed")
	}

	// Parse JSON
	var data CursorData
	if err := json.Unmarshal(decoded, &data); err != nil {
		return nil, fmt.Errorf("invalid cursor: JSON parse failed")
	}

	// Validate required fields
	if data.FetchedAt == "" || data.ID == "" {
		return nil, fmt.Errorf("invalid cursor: missing required fields")
	}

	// Validate timestamp format
	if _, err := time.Parse(time.RFC3339, data.FetchedAt); err != nil {
		return nil, fmt.Errorf("invalid cursor: invalid timestamp format")
	}

	return &data, nil
}

// EncodeCursor creates a base64-encoded JSON cursor
func EncodeCursor(fetchedAt time.Time, id string) string {
	data := CursorData{
		FetchedAt: fetchedAt.Format(time.RFC3339),
		ID:        id,
	}

	jsonData, _ := json.Marshal(data)
	return base64.URLEncoding.EncodeToString(jsonData)
}

// ValidateAndNormalize validates filter parameters and returns error if invalid
func (f *PropertyFilters) ValidateAndNormalize() error {
	// Validate range filters (min <= max)
	if f.MinRent != nil && f.MaxRent != nil && *f.MinRent > *f.MaxRent {
		return fmt.Errorf("min_rent cannot be greater than max_rent")
	}
	if f.MinArea != nil && f.MaxArea != nil && *f.MinArea > *f.MaxArea {
		return fmt.Errorf("min_area cannot be greater than max_area")
	}
	if f.MinBuildingAge != nil && f.MaxBuildingAge != nil && *f.MinBuildingAge > *f.MaxBuildingAge {
		return fmt.Errorf("min_building_age cannot be greater than max_building_age")
	}
	if f.MinFloor != nil && f.MaxFloor != nil && *f.MinFloor > *f.MaxFloor {
		return fmt.Errorf("min_floor cannot be greater than max_floor")
	}

	// Validate array limits (prevent abuse)
	if len(f.FloorPlans) > 20 {
		return fmt.Errorf("floor_plans: maximum 20 items allowed")
	}
	if len(f.BuildingTypes) > 10 {
		return fmt.Errorf("building_types: maximum 10 items allowed")
	}
	if len(f.Facilities) > 30 {
		return fmt.Errorf("facilities: maximum 30 items allowed")
	}
	if len(f.ExcludeIDs) > 500 {
		return fmt.Errorf("exclude_ids: maximum 500 items allowed")
	}

	// Validate sort parameter (whitelist)
	validSorts := map[string]bool{
		"":                  true, // default
		"newest":            true,
		"fetched_at":        true,
		"fetched_at_desc":   true,
		"fetched_at_asc":    true,
		"rent_asc":          true,
		"rent_desc":         true,
		"area_desc":         true,
		"area_asc":          true,
		"walk_time_asc":     true,
		"building_age_asc":  true,
		"building_age_desc": true,
	}
	if !validSorts[f.SortBy] {
		return fmt.Errorf("invalid sort parameter: %s", f.SortBy)
	}

	// Set defaults
	if f.Limit <= 0 {
		f.Limit = 50
	}
	if f.Limit > 20000 {
		f.Limit = 20000
	}

	return nil
}

// GetPropertiesWithSort retrieves all properties with custom sorting
func (gdb *GormDB) GetPropertiesWithSort(sortBy string) ([]models.Property, error) {
	return gdb.GetPropertiesWithFilters(PropertyFilters{SortBy: sortBy})
}

// GetPropertiesWithFilters retrieves properties with filtering and sorting
func (gdb *GormDB) GetPropertiesWithFilters(filters PropertyFilters) ([]models.Property, error) {
	var properties []models.Property

	// Start building query
	query := gdb.db.Model(&models.Property{})

	// Apply station filter (EXISTS on property_stations)
	if filters.Station != "" {
		query = query.Where("EXISTS (SELECT 1 FROM property_stations ps WHERE ps.property_id = properties.id AND ps.station_name LIKE ?)",
			"%"+filters.Station+"%")
	}

	// Apply line filter (EXISTS on property_stations)
	if filters.Line != "" {
		query = query.Where("EXISTS (SELECT 1 FROM property_stations ps WHERE ps.property_id = properties.id AND ps.line_name LIKE ?)",
			"%"+filters.Line+"%")
	}

	// Apply walk time filter
	if filters.MaxWalk > 0 {
		if filters.WalkMode == "any" {
			// Any station within MaxWalk minutes
			query = query.Where("EXISTS (SELECT 1 FROM property_stations ps WHERE ps.property_id = properties.id AND ps.walk_minutes IS NOT NULL AND ps.walk_minutes <= ?)",
				filters.MaxWalk)
		} else {
			// Default: nearest station only (compatibility with legacy walk_time field)
			query = query.Where("walk_time IS NOT NULL AND walk_time <= ?", filters.MaxWalk)
		}
	}

	// Map sort parameter to SQL ORDER BY clause (MySQL syntax)
	// Use CASE to put NULLs last for ASC, first for DESC
	var orderClause string
	switch filters.SortBy {
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

	err := query.Order(orderClause).Find(&properties).Error
	return properties, err
}

// applyFilters applies all WHERE conditions to the query (shared by COUNT and LIST)
func (gdb *GormDB) applyFilters(query *gorm.DB, filters PropertyFilters) *gorm.DB {
	// Default: exclude removed properties
	query = query.Where("status = ?", "active")

	// Station filter (EXISTS on property_stations)
	if filters.Station != "" {
		query = query.Where("EXISTS (SELECT 1 FROM property_stations ps WHERE ps.property_id = properties.id AND ps.station_name LIKE ?)",
			"%"+filters.Station+"%")
	}

	// Line filter (EXISTS on property_stations)
	if filters.Line != "" {
		query = query.Where("EXISTS (SELECT 1 FROM property_stations ps WHERE ps.property_id = properties.id AND ps.line_name LIKE ?)",
			"%"+filters.Line+"%")
	}

	// Walk time filter
	if filters.MaxWalk > 0 {
		if filters.WalkMode == "any" {
			query = query.Where("EXISTS (SELECT 1 FROM property_stations ps WHERE ps.property_id = properties.id AND ps.walk_minutes IS NOT NULL AND ps.walk_minutes <= ?)",
				filters.MaxWalk)
		} else {
			query = query.Where("walk_time IS NOT NULL AND walk_time <= ?", filters.MaxWalk)
		}
	}

	// Rent range filter
	if filters.MinRent != nil {
		query = query.Where("rent >= ?", *filters.MinRent*10000) // Convert 万円 to 円
	}
	if filters.MaxRent != nil {
		query = query.Where("rent <= ?", *filters.MaxRent*10000)
	}

	// Area range filter
	if filters.MinArea != nil {
		query = query.Where("area >= ?", *filters.MinArea)
	}
	if filters.MaxArea != nil {
		query = query.Where("area <= ?", *filters.MaxArea)
	}

	// Building age range filter
	if filters.MinBuildingAge != nil {
		query = query.Where("building_age >= ?", *filters.MinBuildingAge)
	}
	if filters.MaxBuildingAge != nil {
		query = query.Where("building_age <= ?", *filters.MaxBuildingAge)
	}

	// Floor range filter
	if filters.MinFloor != nil {
		query = query.Where("floor >= ?", *filters.MinFloor)
	}
	if filters.MaxFloor != nil {
		query = query.Where("floor <= ?", *filters.MaxFloor)
	}

	// Floor plans filter (multi-select)
	if len(filters.FloorPlans) > 0 {
		query = query.Where("floor_plan IN ?", filters.FloorPlans)
	}

	// Building types filter (multi-select)
	if len(filters.BuildingTypes) > 0 {
		query = query.Where("building_type IN ?", filters.BuildingTypes)
	}

	// Facilities filter (JSON array contains - requires MySQL JSON functions or simple LIKE)
	if len(filters.Facilities) > 0 {
		for _, facility := range filters.Facilities {
			query = query.Where("facilities LIKE ?", "%\""+facility+"\"%")
		}
	}

	// Exclude IDs filter
	if len(filters.ExcludeIDs) > 0 {
		query = query.Where("id NOT IN ?", filters.ExcludeIDs)
	}

	return query
}

// GetPropertiesWithFiltersPaginated retrieves properties with filtering, sorting, and pagination
func (gdb *GormDB) GetPropertiesWithFiltersPaginated(filters PropertyFilters) (*PaginatedPropertiesResponse, error) {
	// Validate filters
	if err := filters.ValidateAndNormalize(); err != nil {
		return nil, err
	}

	// Build base query for COUNT (no cursor condition)
	countQuery := gdb.db.Model(&models.Property{})
	countQuery = gdb.applyFilters(countQuery, filters)

	// Get total count BEFORE pagination (cursor does NOT affect total)
	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, err
	}

	// Build query for LIST (with cursor condition if present)
	listQuery := gdb.db.Model(&models.Property{})
	listQuery = gdb.applyFilters(listQuery, filters)

	// Map sort parameter to SQL ORDER BY clause (MySQL syntax)
	// Use CASE to put NULLs last for ASC, first for DESC
	var orderClause string
	isCursorCompatible := false // Only newest/fetched_at sorts support cursor

	switch filters.SortBy {
	case "newest", "fetched_at", "fetched_at_desc", "":
		orderClause = "fetched_at DESC, id DESC"
		isCursorCompatible = true
	case "fetched_at_asc":
		orderClause = "fetched_at ASC, id ASC"
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
		orderClause = "fetched_at DESC, id DESC"
		isCursorCompatible = true
	}

	// Apply cursor condition for cursor-compatible sorts
	if filters.Cursor != "" && isCursorCompatible {
		cursorData, err := DecodeCursor(filters.Cursor)
		if err != nil {
			return nil, err
		}

		// Parse timestamp
		cursorTime, err := time.Parse(time.RFC3339, cursorData.FetchedAt)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor timestamp")
		}

		// Apply cursor condition: (fetched_at < cursor_time) OR (fetched_at = cursor_time AND id < cursor_id)
		listQuery = listQuery.Where(
			"(fetched_at < ?) OR (fetched_at = ? AND id < ?)",
			cursorTime, cursorTime, cursorData.ID,
		)
	}

	// Apply pagination
	var properties []models.Property
	offset := 0

	// Use offset only if cursor is not present (legacy compatibility)
	if filters.Cursor == "" && filters.Offset != nil {
		offset = *filters.Offset
	}

	err := listQuery.Order(orderClause).
		Limit(filters.Limit).
		Offset(offset).
		Find(&properties).Error

	if err != nil {
		return nil, err
	}

	// Generate next_cursor for cursor-compatible sorts
	var nextCursor string
	if isCursorCompatible && len(properties) == filters.Limit {
		lastProperty := properties[len(properties)-1]
		nextCursor = EncodeCursor(lastProperty.FetchedAt, lastProperty.ID)
	}

	return &PaginatedPropertiesResponse{
		Properties: properties,
		Total:      total,
		Limit:      filters.Limit,
		Offset:     offset,
		NextCursor: nextCursor,
	}, nil
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

// GetPropertyStations retrieves all stations for a property
func (gdb *GormDB) GetPropertyStations(propertyID string) ([]models.PropertyStation, error) {
	var stations []models.PropertyStation
	err := gdb.db.Where("property_id = ?", propertyID).Order("sort_order ASC").Find(&stations).Error
	return stations, err
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

// SavePropertyWithStationsAndImages saves a property with its stations and images in a transaction
func (gdb *GormDB) SavePropertyWithStationsAndImages(p *models.Property, stations []models.PropertyStation, images []models.PropertyImage) error {
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

	// Use transaction to save property, stations, and images
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
			// Update existing property
			p.ID = existing.ID // Preserve existing ID
			p.CreatedAt = existing.CreatedAt // Preserve creation time
			if err := tx.Save(p).Error; err != nil {
				return err
			}
		}

		// Delete existing stations for this property
		if err := tx.Where("property_id = ?", p.ID).Delete(&models.PropertyStation{}).Error; err != nil {
			return err
		}

		// Insert new stations
		if len(stations) > 0 {
			if err := tx.Create(&stations).Error; err != nil {
				return err
			}
		}

		// Delete existing images for this property
		if err := tx.Where("property_id = ?", p.ID).Delete(&models.PropertyImage{}).Error; err != nil {
			return err
		}

		// Insert new images
		if len(images) > 0 {
			if err := tx.Create(&images).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// GetPropertyImages retrieves all images for a property
func (gdb *GormDB) GetPropertyImages(propertyID string) ([]models.PropertyImage, error) {
	var images []models.PropertyImage
	err := gdb.db.Where("property_id = ?", propertyID).Order("sort_order ASC").Find(&images).Error
	return images, err
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
