package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"real-estate-portal/internal/config"
	"real-estate-portal/internal/database"
	"real-estate-portal/internal/handlers"
	"real-estate-portal/internal/models"
	"real-estate-portal/internal/ratelimit"
	"real-estate-portal/internal/scheduler"
	"real-estate-portal/internal/scraper"
	"real-estate-portal/internal/search"
	"real-estate-portal/internal/snapshot"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var (
	db              *database.DB
	gormDB          *database.GormDB
	searchClient    *search.SearchClient
	appConfig       *config.Config
	rateLimiter     *ratelimit.RateLimiter
	appScheduler    *scheduler.Scheduler
	queueWorker     *scheduler.QueueWorker
	snapshotService *snapshot.Service
)

func main() {
	// Load configuration
	configPath := getEnv("CONFIG_PATH", "/app/config/scraper_config.yaml")
	var err error
	appConfig, err = config.LoadConfig(configPath)
	if err != nil {
		log.Printf("Warning: Failed to load config from %s: %v. Using defaults.", configPath, err)
		appConfig = config.DefaultConfig()
	} else {
		log.Printf("Loaded configuration from %s", configPath)
	}

	// Initialize database based on configuration
	dbType := appConfig.Database.Type
	if dbType == "" {
		dbType = getEnv("DB_TYPE", "postgres")
	}

	if dbType == "mysql" {
		log.Println("Using MySQL with GORM")
		mysqlCfg := appConfig.Database.MySQL

		// Get port as string, handle 0 as empty
		portStr := ""
		if mysqlCfg.Port > 0 {
			portStr = fmt.Sprintf("%d", mysqlCfg.Port)
		}

		gormDB, err = database.NewGormDB(
			getEnvOrConfig(mysqlCfg.Host, "DB_HOST", "mysql"),
			getEnvOrConfig(portStr, "DB_PORT", "3306"),
			getEnvOrConfig(mysqlCfg.User, "DB_USER", "realestate_user"),
			getEnvOrConfig(mysqlCfg.Password, "DB_PASSWORD", "realestate_pass"),
			getEnvOrConfig(mysqlCfg.Database, "DB_NAME", "realestate_db"),
		)
		if err != nil {
			log.Fatalf("Failed to connect to MySQL: %v", err)
		}
		defer gormDB.Close()

		// Initialize schema with GORM AutoMigrate
		if err := gormDB.InitSchema(); err != nil {
			log.Fatalf("Failed to initialize schema: %v", err)
		}
	} else {
		log.Println("Using PostgreSQL")
		pgCfg := appConfig.Database.Postgres

		// Get port as string, handle 0 as empty
		portStr := ""
		if pgCfg.Port > 0 {
			portStr = fmt.Sprintf("%d", pgCfg.Port)
		}

		db, err = database.NewDB(
			getEnvOrConfig(pgCfg.Host, "DB_HOST", "db"),
			getEnvOrConfig(portStr, "DB_PORT", "5432"),
			getEnvOrConfig(pgCfg.User, "DB_USER", "realestate_user"),
			getEnvOrConfig(pgCfg.Password, "DB_PASSWORD", "realestate_pass"),
			getEnvOrConfig(pgCfg.Database, "DB_NAME", "realestate_db"),
		)
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}
		defer db.Close()

		// Initialize schema
		if err := db.InitSchema(); err != nil {
			log.Fatalf("Failed to initialize schema: %v", err)
		}
	}

	// Initialize Meilisearch using config
	meilisearchHost := appConfig.Search.Meilisearch.Host
	if meilisearchHost == "" {
		meilisearchHost = getEnv("MEILISEARCH_HOST", "http://meilisearch:7700")
	}
	meilisearchKey := appConfig.Search.Meilisearch.APIKey
	if meilisearchKey == "" {
		meilisearchKey = getEnv("MEILISEARCH_KEY", "masterKey123")
	}

	searchClient = search.NewSearchClient(meilisearchHost, meilisearchKey)

	// Wait for Meilisearch to be ready
	time.Sleep(2 * time.Second)

	if err := searchClient.InitIndex(); err != nil {
		log.Printf("Warning: Failed to initialize search index: %v", err)
	}

	// Initialize rate limiter
	rateLimiter = ratelimit.NewRateLimiter(
		appConfig.RateLimit.RequestsPerMinute,
		appConfig.RateLimit.RequestsPerHour,
		appConfig.Scraper.MaxRequestsPerDay,
		appConfig.RateLimit.Enabled,
	)
	log.Printf("Rate limiter initialized: %d req/min, %d req/hour, %d req/day (enabled: %v)",
		appConfig.RateLimit.RequestsPerMinute,
		appConfig.RateLimit.RequestsPerHour,
		appConfig.Scraper.MaxRequestsPerDay,
		appConfig.RateLimit.Enabled,
	)

	// Initialize snapshot service (MySQL only)
	if gormDB != nil {
		sqlDB, _ := gormDB.GetDB()
		snapshotService = snapshot.NewService(sqlDB)
		log.Println("Snapshot service initialized")
	}

	// Initialize and start scheduler (MySQL only)
	if gormDB != nil {
		sqlDB, _ := gormDB.GetDB()
		appScheduler = scheduler.NewScheduler(sqlDB, appConfig)
		if err := appScheduler.Start(); err != nil {
			log.Printf("Warning: Failed to start scheduler: %v", err)
		}
		defer appScheduler.Stop()

		// Initialize and start queue worker
		queueWorker = scheduler.NewQueueWorker(sqlDB)
		queueWorker.Start()
		defer queueWorker.Stop()
		log.Println("Queue worker started")
	}

	// Setup Gin router
	r := gin.Default()

	// CORS configuration
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5176"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		AllowCredentials: true,
	}))

	// Routes
	r.GET("/health", healthCheck)
	r.GET("/api/properties", getProperties)
	r.GET("/api/properties/:id", getProperty)

	// Scraping routes with rate limiting
	r.POST("/api/scrape", rateLimitMiddleware(), scrapeURL)
	r.POST("/api/scrape/batch", rateLimitMiddleware(), scrapeBatch)
	r.POST("/api/scrape/list", rateLimitMiddleware(), scrapeListPage)
	r.POST("/api/scrape/update", rateLimitMiddleware(), scrapeAndUpdate)

	// Rate limiter stats endpoint
	r.GET("/api/ratelimit/stats", getRateLimitStats)

	// Scheduler and snapshot endpoints
	r.POST("/api/scheduler/run", triggerScheduledScraping)
	r.GET("/api/properties/:id/history", getPropertyHistory)
	r.GET("/api/changes/recent", getRecentChanges)

	// Queue worker stats endpoint
	r.GET("/api/queue/stats", getQueueStats)

	r.GET("/api/search", searchProperties)
	r.POST("/api/search/advanced", advancedSearchProperties)
	r.GET("/api/search/facets", getSearchFacets)
	r.POST("/api/search/reindex", reindexAllProperties)
	r.GET("/api/filter", filterProperties)

	// Admin API routes (requires authentication in production)
	if gormDB != nil {
		sqlDB, _ := gormDB.GetDB()
		adminHandler := handlers.NewAdminHandler(sqlDB, appScheduler)

		admin := r.Group("/api/admin")
		{
			// Statistics
			admin.GET("/stats", adminHandler.GetStats)
			admin.GET("/activity", adminHandler.GetRecentActivity)
			admin.GET("/area-stats", adminHandler.GetAreaStats)
			admin.GET("/price-distribution", adminHandler.GetPriceDistribution)

			// Scraping control
			admin.POST("/scraping/trigger", adminHandler.TriggerScraping)
			admin.GET("/scraping/status", adminHandler.GetScrapingStatus)

			// Cleanup operations
			admin.POST("/cleanup/run", adminHandler.RunCleanup)
			admin.GET("/cleanup/logs", adminHandler.GetDeleteLogs)

			// Property history
			admin.GET("/properties/:id/history", adminHandler.GetPropertyHistory)
			admin.GET("/changes/recent", adminHandler.GetRecentChanges)
		}

		log.Println("Admin API routes registered at /api/admin/*")
	}

	port := getEnv("PORT", "8084")
	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"time":   time.Now(),
	})
}

func getProperties(c *gin.Context) {
	// Build filters from query parameters
	filters := database.PropertyFilters{
		Station:  c.Query("station"),
		Line:     c.Query("line"),
		WalkMode: c.DefaultQuery("walk_mode", "nearest"),
		SortBy:   c.DefaultQuery("sort", "fetched_at"),
		Cursor:   c.Query("cursor"),
	}

	// Parse numeric parameters
	if maxWalkStr := c.Query("max_walk"); maxWalkStr != "" {
		if maxWalk, parseErr := strconv.Atoi(maxWalkStr); parseErr == nil && maxWalk > 0 {
			filters.MaxWalk = maxWalk
		}
	}

	// Rent range
	if minRentStr := c.Query("min_rent"); minRentStr != "" {
		if minRent, parseErr := strconv.Atoi(minRentStr); parseErr == nil {
			filters.MinRent = &minRent
		}
	}
	if maxRentStr := c.Query("max_rent"); maxRentStr != "" {
		if maxRent, parseErr := strconv.Atoi(maxRentStr); parseErr == nil {
			filters.MaxRent = &maxRent
		}
	}

	// Area range
	if minAreaStr := c.Query("min_area"); minAreaStr != "" {
		if minArea, parseErr := strconv.ParseFloat(minAreaStr, 64); parseErr == nil {
			filters.MinArea = &minArea
		}
	}
	if maxAreaStr := c.Query("max_area"); maxAreaStr != "" {
		if maxArea, parseErr := strconv.ParseFloat(maxAreaStr, 64); parseErr == nil {
			filters.MaxArea = &maxArea
		}
	}

	// Building age range
	if minAgeStr := c.Query("min_building_age"); minAgeStr != "" {
		if minAge, parseErr := strconv.Atoi(minAgeStr); parseErr == nil {
			filters.MinBuildingAge = &minAge
		}
	}
	if maxAgeStr := c.Query("max_building_age"); maxAgeStr != "" {
		if maxAge, parseErr := strconv.Atoi(maxAgeStr); parseErr == nil {
			filters.MaxBuildingAge = &maxAge
		}
	}

	// Floor range
	if minFloorStr := c.Query("min_floor"); minFloorStr != "" {
		if minFloor, parseErr := strconv.Atoi(minFloorStr); parseErr == nil {
			filters.MinFloor = &minFloor
		}
	}
	if maxFloorStr := c.Query("max_floor"); maxFloorStr != "" {
		if maxFloor, parseErr := strconv.Atoi(maxFloorStr); parseErr == nil {
			filters.MaxFloor = &maxFloor
		}
	}

	// Multi-select filters (comma-separated)
	if floorPlansStr := c.Query("floor_plans"); floorPlansStr != "" {
		filters.FloorPlans = strings.Split(floorPlansStr, ",")
	}
	if buildingTypesStr := c.Query("building_types"); buildingTypesStr != "" {
		filters.BuildingTypes = strings.Split(buildingTypesStr, ",")
	}
	if facilitiesStr := c.Query("facilities"); facilitiesStr != "" {
		filters.Facilities = strings.Split(facilitiesStr, ",")
	}
	if excludeIDsStr := c.Query("exclude_ids"); excludeIDsStr != "" {
		filters.ExcludeIDs = strings.Split(excludeIDsStr, ",")
	}

	// Pagination parameters
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, parseErr := strconv.Atoi(limitStr); parseErr == nil && limit > 0 {
			filters.Limit = limit
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, parseErr := strconv.Atoi(offsetStr); parseErr == nil && offset >= 0 {
			filters.Offset = &offset
		}
	}

	// Cursor parameter (for cursor-based pagination)
	if cursor := c.Query("cursor"); cursor != "" {
		filters.Cursor = cursor
	}

	// Always use paginated endpoint with GORM
	if gormDB != nil {
		start := time.Now()
		result, err := gormDB.GetPropertiesWithFiltersPaginated(filters)
		duration := time.Since(start)

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Log search API performance for monitoring
		log.Printf("[Search API] duration_ms=%d total=%d limit=%d has_cursor=%v sort=%s",
			duration.Milliseconds(), result.Total, result.Limit, filters.Cursor != "", filters.SortBy)

		c.JSON(http.StatusOK, result)
		return
	}

	// Legacy fallback (should not be reached in production)
	properties, err := db.GetPropertiesWithSort(filters.SortBy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, properties)
}

func getProperty(c *gin.Context) {
	id := c.Param("id")
	var property *models.Property
	var err error

	if gormDB != nil {
		property, err = gormDB.GetPropertyByID(id)
	} else {
		property, err = db.GetPropertyByID(id)
	}

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Property not found"})
		return
	}

	// Fetch stations and images if using GORM
	var stations []models.PropertyStation
	var images []models.PropertyImage
	if gormDB != nil {
		stations, _ = gormDB.GetPropertyStations(id)
		images, _ = gormDB.GetPropertyImages(id)
	}

	// Create response with stations and images
	response := gin.H{
		"property": property,
		"stations": stations,
		"images":   images,
	}

	c.JSON(http.StatusOK, response)
}

// createScraper creates a new scraper instance with configuration
func createScraper() *scraper.Scraper {
	if appConfig == nil {
		return scraper.NewScraper()
	}

	return scraper.NewScraperWithConfig(scraper.ScraperConfig{
		Timeout:      appConfig.Scraper.GetTimeout(),
		MaxRetries:   appConfig.Scraper.MaxRetries,
		RetryDelay:   appConfig.Scraper.GetRetryDelay(),
		RequestDelay: appConfig.Scraper.GetRequestDelay(),
	})
}

func scrapeURL(c *gin.Context) {
	var req struct {
		URL string `json:"url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Apply DetailLimiter for single property scraping (5 per hour max)
	scraper.DetailLimiter.Acquire("single")

	// Scrape the property
	s := createScraper()
	property, err := s.ScrapeProperty(req.URL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Save to database with stations and images (if using GORM)
	if gormDB != nil {
		// Get stations and images from scraper and convert to models
		stations := s.GetLastStationsAsModels(property.ID)
		images := s.GetLastImagesAsModels(property.ID)
		err = gormDB.SavePropertyWithStationsAndImages(property, stations, images)

		// Log station and image save operation
		if err == nil {
			if len(stations) == 0 {
				log.Printf("[stations] property_id=%s stations_len=0 skip_delete_preserve_existing", property.ID)
			} else {
				log.Printf("[stations] property_id=%s stations_len=%d saved", property.ID, len(stations))
			}
			if len(images) > 0 {
				log.Printf("[images] property_id=%s images_len=%d saved", property.ID, len(images))
			}
		}
	} else {
		// Legacy database doesn't support stations
		err = db.SaveProperty(property)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Index in Meilisearch
	if err := searchClient.IndexProperty(property); err != nil {
		log.Printf("Warning: Failed to index property: %v", err)
	}

	c.JSON(http.StatusOK, property)
}

func scrapeBatch(c *gin.Context) {
	var req struct {
		URLs []string `json:"urls" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s := createScraper()
	var properties []models.Property
	var errors []string

	for _, url := range req.URLs {
		property, err := s.ScrapeProperty(url)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", url, err))
			continue
		}

		if gormDB != nil {
			err = gormDB.SaveProperty(property)
		} else {
			err = db.SaveProperty(property)
		}

		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", url, err))
			continue
		}

		properties = append(properties, *property)

		// Small delay to be respectful
		time.Sleep(1 * time.Second)
	}

	// Index all properties
	if len(properties) > 0 {
		if err := searchClient.IndexProperties(properties); err != nil {
			log.Printf("Warning: Failed to index properties: %v", err)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": len(properties),
		"failed":  len(errors),
		"errors":  errors,
		"properties": properties,
	})
}

func scrapeListPage(c *gin.Context) {
	var req struct {
		URL         string `json:"url" binding:"required"`
		Limit       int    `json:"limit"`       // Optional: max number of properties to scrape
		Concurrency int    `json:"concurrency"` // Optional: number of concurrent scrapers (default: 5)
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Default limit to 20 if not specified
	if req.Limit == 0 {
		req.Limit = 20
	}

	// Default concurrency to 5
	if req.Concurrency == 0 {
		req.Concurrency = 5
	}

	s := createScraper()

	// Step 1: Extract property URLs from list page
	log.Printf("Scraping list page: %s", req.URL)
	propertyURLs, err := s.ScrapeListPage(req.URL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to scrape list page: %v", err)})
		return
	}

	log.Printf("Found %d property URLs", len(propertyURLs))

	// Apply limit
	if len(propertyURLs) > req.Limit {
		propertyURLs = propertyURLs[:req.Limit]
	}

	// Step 2: Check which properties already exist (differential scraping)
	log.Printf("Checking for existing properties...")
	existingURLs := make(map[string]bool)
	newURLs := []string{}

	for _, url := range propertyURLs {
		// Extract Yahoo property ID from URL for efficient lookup
		normalizedURL := normalizeURLForCheck(url)
		parts := strings.Split(normalizedURL, "/detail/")
		if len(parts) == 2 {
			propertyID := strings.Split(parts[1], "?")[0]
			propertyID = strings.TrimSuffix(propertyID, "/")

			// Check if property exists with this source_property_id
			var count int64
			if gormDB != nil {
				gormDB.DB().Model(&models.Property{}).Where("source = ? AND source_property_id = ?", "yahoo", propertyID).Count(&count)
			}
			if count > 0 {
				existingURLs[url] = true
				// Update last_seen_at for existing property
				if gormDB != nil {
					gormDB.DB().Model(&models.Property{}).
						Where("source = ? AND source_property_id = ?", "yahoo", propertyID).
						Update("last_seen_at", time.Now())
				}
			} else {
				newURLs = append(newURLs, url)
			}
		} else {
			// If URL doesn't match expected format, treat as new
			newURLs = append(newURLs, url)
		}
	}

	log.Printf("Found %d existing properties (last_seen_at updated), %d new properties to scrape", len(existingURLs), len(newURLs))

	// Step 2.5: Add new properties to the detail_scrape_queue
	if gormDB != nil && len(newURLs) > 0 {
		for _, url := range newURLs {
			// Extract property ID from URL
			normalizedURL := normalizeURLForCheck(url)
			parts := strings.Split(normalizedURL, "/detail/")
			if len(parts) == 2 {
				propertyID := strings.Split(parts[1], "?")[0]
				propertyID = strings.TrimSuffix(propertyID, "/")

				// Add to queue with proper failed handling
				var existing models.DetailScrapeQueue
				err := gormDB.DB().Where("source = ? AND source_property_id = ?", "yahoo", propertyID).
					First(&existing).Error

				if errors.Is(err, gorm.ErrRecordNotFound) {
					// No existing record - create new
					queue := models.DetailScrapeQueue{
						Source:           "yahoo",
						SourcePropertyID: propertyID,
						DetailURL:        normalizedURL,
						Status:           models.QueueStatusPending,
						Priority:         0,
						Attempts:         0,
					}
					if createErr := gormDB.DB().Create(&queue).Error; createErr != nil {
						log.Printf("Warning: Failed to create queue for %s: %v", propertyID, createErr)
					}
				} else if err != nil {
					log.Printf("Warning: Failed to check queue for %s: %v", propertyID, err)
				} else {
					// Record exists - handle based on status
					switch existing.Status {
					case models.QueueStatusPending, models.QueueStatusProcessing:
						// Already in queue, do nothing
					case models.QueueStatusFailed:
						// Retry failed items by resetting to pending
						updates := map[string]interface{}{
							"status":        models.QueueStatusPending,
							"attempts":      0,
							"last_error":    "",
							"next_retry_at": nil,
						}
						if updateErr := gormDB.DB().Model(&existing).Updates(updates).Error; updateErr != nil {
							log.Printf("Warning: Failed to reset failed queue for %s: %v", propertyID, updateErr)
						}
					case models.QueueStatusPermanentFail:
						// Don't retry permanent failures (404, etc)
					case models.QueueStatusDone:
						// Already successfully scraped, do nothing
					}
				}
			}
		}
		log.Printf("✅ Added %d new properties to detail_scrape_queue", len(newURLs))
	}

	// Step 3: QUEUE-ONLY MODE - Do NOT process immediately
	// Detail scraping should ONLY happen via queue workers/scheduler
	// This prevents accidental bursts and keeps rate limiting centralized
	log.Printf("✅ List scraping complete. URLs added to queue for processing by scheduler/worker.")

	// Count queue status for response
	var queueStats struct {
		Pending    int64
		Processing int64
		Done       int64
		Failed     int64
	}
	if gormDB != nil {
		gormDB.DB().Model(&models.DetailScrapeQueue{}).Where("status = ?", models.QueueStatusPending).Count(&queueStats.Pending)
		gormDB.DB().Model(&models.DetailScrapeQueue{}).Where("status = ?", models.QueueStatusProcessing).Count(&queueStats.Processing)
		gormDB.DB().Model(&models.DetailScrapeQueue{}).Where("status = ?", models.QueueStatusDone).Count(&queueStats.Done)
		gormDB.DB().Model(&models.DetailScrapeQueue{}).Where("status = ?", models.QueueStatusFailed).Count(&queueStats.Failed)
	}

	// NOTE: Removed immediate detail scraping from here
	// All detail scraping now happens via:
	// - POST /api/scrape (single, with DetailLimiter)
	// - Scheduler/worker (processes queue with rate limits)

	// Return queue-only response
	c.JSON(http.StatusOK, gin.H{
		"message":         "List page scraped successfully. URLs added to queue.",
		"urls_found":      len(propertyURLs),
		"existing":        len(existingURLs),
		"new_to_queue":    len(newURLs),
		"queue_status": gin.H{
			"pending":    queueStats.Pending,
			"processing": queueStats.Processing,
			"done":       queueStats.Done,
			"failed":     queueStats.Failed,
		},
	})
}

// REMOVED: Immediate detail scraping logic
// All detail scraping moved to queue worker/scheduler only

func scrapeAndUpdate(c *gin.Context) {
	var req struct {
		URL   string `json:"url" binding:"required"`
		Limit int    `json:"limit"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Default limit
	if req.Limit == 0 {
		req.Limit = 50
	}

	log.Printf("Starting differential update for: %s", req.URL)

	s := createScraper()

	// Step 1: Extract property URLs from list page
	propertyURLs, err := s.ScrapeListPage(req.URL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to scrape list page: %v", err)})
		return
	}

	log.Printf("Found %d property URLs", len(propertyURLs))

	// Apply limit
	if len(propertyURLs) > req.Limit {
		propertyURLs = propertyURLs[:req.Limit]
	}

	// Step 2: Scrape each property
	var scrapedProperties []models.Property
	var scrapeErrors []string
	var permanentFailures []string

	for i, url := range propertyURLs {
		log.Printf("Scraping property %d/%d: %s", i+1, len(propertyURLs), url)

		property, err := s.ScrapeProperty(url)
		if err != nil {
			errMsg := err.Error()

			// Check for permanent failure (404)
			if strings.Contains(errMsg, "permanent_fail") || strings.Contains(errMsg, "404") {
				log.Printf("Permanent failure (404) for %s - not retrying", url)
				permanentFailures = append(permanentFailures, fmt.Sprintf("%s: 404 Not Found (permanent)", url))
				continue
			}

			// Other errors (WAF, timeout, etc.)
			scrapeErrors = append(scrapeErrors, fmt.Sprintf("%s: %v", url, err))
			continue
		}

		scrapedProperties = append(scrapedProperties, *property)
		time.Sleep(2 * time.Second)
	}

	log.Printf("Successfully scraped %d properties", len(scrapedProperties))

	// Step 3: Detect differences (only for GORM/MySQL)
	if gormDB == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Differential update requires MySQL/GORM"})
		return
	}

	newIDs, removedIDs, updatedProperties, err := gormDB.DetectDifferences(scrapedProperties)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to detect differences: %v", err)})
		return
	}

	log.Printf("Differences detected - New: %d, Removed: %d, Updated: %d", len(newIDs), len(removedIDs), len(updatedProperties))

	// Step 4: Apply changes
	var saveErrors []string

	// Mark removed properties
	if len(removedIDs) > 0 {
		if err := gormDB.MarkPropertiesAsRemoved(removedIDs); err != nil {
			saveErrors = append(saveErrors, fmt.Sprintf("Failed to mark properties as removed: %v", err))
		} else {
			log.Printf("Marked %d properties as removed", len(removedIDs))
		}
	}

	// Save new and updated properties
	for _, property := range scrapedProperties {
		if err := gormDB.SaveProperty(&property); err != nil {
			saveErrors = append(saveErrors, fmt.Sprintf("%s: %v", property.ID, err))
			continue
		}
	}

	// Step 5: Update search index
	if len(scrapedProperties) > 0 {
		if err := searchClient.IndexProperties(scrapedProperties); err != nil {
			log.Printf("Warning: Failed to index properties: %v", err)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"scraped":           len(scrapedProperties),
		"new":               len(newIDs),
		"removed":           len(removedIDs),
		"updated":           len(updatedProperties),
		"scrapeErrors":      scrapeErrors,
		"permanentFailures": permanentFailures,
		"saveErrors":        saveErrors,
	})
}

func searchProperties(c *gin.Context) {
	query := c.Query("q")
	limitStr := c.DefaultQuery("limit", "20")

	limit, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil {
		limit = 20
	}

	// If no query, get all from database
	if query == "" {
		var properties []models.Property
		var err error

		if gormDB != nil {
			properties, err = gormDB.GetAllProperties()
		} else {
			properties, err = db.GetAllProperties()
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, properties)
		return
	}

	// Search using Meilisearch
	properties, err := searchClient.Search(query, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, properties)
}

func filterProperties(c *gin.Context) {
	query := c.Query("q")
	limitStr := c.DefaultQuery("limit", "20")

	limit, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil {
		limit = 20
	}

	// Parse filter parameters
	params := search.FilterParams{
		Query: query,
		Limit: limit,
	}

	// Rent range
	if minRentStr := c.Query("min_rent"); minRentStr != "" {
		if minRent, err := strconv.Atoi(minRentStr); err == nil {
			params.MinRent = &minRent
		}
	}
	if maxRentStr := c.Query("max_rent"); maxRentStr != "" {
		if maxRent, err := strconv.Atoi(maxRentStr); err == nil {
			params.MaxRent = &maxRent
		}
	}

	// Floor plans
	if floorPlans := c.QueryArray("floor_plan"); len(floorPlans) > 0 {
		params.FloorPlans = floorPlans
	}

	// Max walk time
	if maxWalkStr := c.Query("max_walk_time"); maxWalkStr != "" {
		if maxWalk, err := strconv.Atoi(maxWalkStr); err == nil {
			params.MaxWalkTime = &maxWalk
		}
	}

	// Sort by
	if sortBy := c.Query("sort_by"); sortBy != "" {
		params.SortBy = sortBy
	}

	// If no query and no filters, get all from database
	if query == "" && params.MinRent == nil && params.MaxRent == nil &&
		len(params.FloorPlans) == 0 && params.MaxWalkTime == nil {
		var properties []models.Property
		var err error

		if gormDB != nil {
			properties, err = gormDB.GetAllProperties()
		} else {
			properties, err = db.GetAllProperties()
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, properties)
		return
	}

	// Search with filters using Meilisearch
	properties, err := searchClient.FilterSearch(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, properties)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvOrConfig returns config value if set, otherwise falls back to environment variable, then default
func getEnvOrConfig(configValue, envKey, defaultValue string) string {
	if configValue != "" {
		return configValue
	}
	return getEnv(envKey, defaultValue)
}

// Utility function to load URLs from file
func loadURLsFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var urls []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			urls = append(urls, line)
		}
	}

	return urls, scanner.Err()
}

// rateLimitMiddleware returns a Gin middleware that enforces rate limiting
func rateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rateLimiter.AllowRequest() {
			stats := rateLimiter.GetStats()
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Rate limit exceeded",
				"message": "Too many requests. Please try again later.",
				"stats":   stats,
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// getRateLimitStats returns current rate limiter statistics
func getRateLimitStats(c *gin.Context) {
	stats := rateLimiter.GetStats()
	c.JSON(http.StatusOK, stats)
}

// getQueueStats returns current queue worker statistics
func getQueueStats(c *gin.Context) {
	if queueWorker == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Queue worker is not available (requires MySQL/GORM)",
		})
		return
	}

	stats := queueWorker.GetQueueStats()
	c.JSON(http.StatusOK, stats)
}

// triggerScheduledScraping manually triggers the scheduled scraping job
func triggerScheduledScraping(c *gin.Context) {
	if appScheduler == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Scheduler is not available (requires MySQL/GORM)",
		})
		return
	}

	// Run in background to avoid timeout
	go func() {
		if err := appScheduler.RunNow(); err != nil {
			log.Printf("Manual scraping failed: %v", err)
		}
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Scheduled scraping job started in background",
		"status":  "running",
	})
}

// getPropertyHistory retrieves snapshot history for a property
func getPropertyHistory(c *gin.Context) {
	if snapshotService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Snapshot service is not available (requires MySQL/GORM)",
		})
		return
	}

	propertyID := c.Param("id")
	limitStr := c.DefaultQuery("limit", "30")
	limit, _ := strconv.Atoi(limitStr)

	snapshots, err := snapshotService.GetPropertyHistory(propertyID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"property_id": propertyID,
		"count":       len(snapshots),
		"snapshots":   snapshots,
	})
}

// getRecentChanges retrieves recent property changes
func getRecentChanges(c *gin.Context) {
	if snapshotService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Snapshot service is not available (requires MySQL/GORM)",
		})
		return
	}

	limitStr := c.DefaultQuery("limit", "50")
	limit, _ := strconv.Atoi(limitStr)

	changes, err := snapshotService.GetRecentChanges(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count":   len(changes),
		"changes": changes,
	})
}

// advancedSearchProperties performs advanced search with filters and facets
func advancedSearchProperties(c *gin.Context) {
	var reqBody struct {
		Query       string   `json:"query"`
		Limit       int64    `json:"limit"`
		Offset      int64    `json:"offset"`
		MinRent     *int     `json:"min_rent"`
		MaxRent     *int     `json:"max_rent"`
		FloorPlans  []string `json:"floor_plans"`
		MinArea     *float64 `json:"min_area"`
		MaxArea     *float64 `json:"max_area"`
		MaxWalkTime *int     `json:"max_walk_time"`
		Sort        string   `json:"sort"` // "rent_asc", "rent_desc", "area_desc", etc.
		Facets      []string `json:"facets"`
	}

	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build filter conditions
	filters := []string{}

	if reqBody.MinRent != nil {
		filters = append(filters, fmt.Sprintf("rent >= %d", *reqBody.MinRent))
	}
	if reqBody.MaxRent != nil {
		filters = append(filters, fmt.Sprintf("rent <= %d", *reqBody.MaxRent))
	}
	if reqBody.MinArea != nil {
		filters = append(filters, fmt.Sprintf("area >= %f", *reqBody.MinArea))
	}
	if reqBody.MaxArea != nil {
		filters = append(filters, fmt.Sprintf("area <= %f", *reqBody.MaxArea))
	}
	if reqBody.MaxWalkTime != nil {
		filters = append(filters, fmt.Sprintf("walk_time <= %d", *reqBody.MaxWalkTime))
	}
	if len(reqBody.FloorPlans) > 0 {
		planFilters := make([]string, len(reqBody.FloorPlans))
		for i, plan := range reqBody.FloorPlans {
			planFilters[i] = fmt.Sprintf("floor_plan = '%s'", plan)
		}
		filters = append(filters, "("+strings.Join(planFilters, " OR ")+")")
	}

	// Build sort conditions
	sortConditions := []string{}
	if reqBody.Sort != "" {
		switch reqBody.Sort {
		case "rent_asc":
			sortConditions = append(sortConditions, "rent:asc")
		case "rent_desc":
			sortConditions = append(sortConditions, "rent:desc")
		case "area_desc":
			sortConditions = append(sortConditions, "area:desc")
		case "walk_time_asc":
			sortConditions = append(sortConditions, "walk_time:asc")
		case "building_age_asc":
			sortConditions = append(sortConditions, "building_age:asc")
		case "newest":
			sortConditions = append(sortConditions, "created_at:desc")
		}
	}

	// Default facets
	facets := reqBody.Facets
	if len(facets) == 0 {
		facets = []string{"floor_plan", "station"}
	}

	// Perform search
	searchReq := search.SearchRequest{
		Query:        reqBody.Query,
		Limit:        reqBody.Limit,
		Offset:       reqBody.Offset,
		Filter:       filters,
		Sort:         sortConditions,
		FacetsFilter: facets,
	}

	if searchReq.Limit == 0 {
		searchReq.Limit = 20
	}

	result, err := searchClient.AdvancedSearch(searchReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"hits":            result.Hits,
		"total_hits":      result.TotalHits,
		"facets":          result.Facets,
		"processing_time": result.ProcessingTime,
		"query":           reqBody.Query,
		"filters":         filters,
	})
}

// getSearchFacets retrieves facet distributions
func getSearchFacets(c *gin.Context) {
	facetsParam := c.DefaultQuery("facets", "floor_plan,station")
	facets := strings.Split(facetsParam, ",")

	facetDist, err := searchClient.GetFacets(facets)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"facets": facetDist,
	})
}

// reindexAllProperties re-indexes all properties from database to Meilisearch
func reindexAllProperties(c *gin.Context) {
	log.Println("[Reindex] Starting full reindex of all properties")

	// Get all properties from database
	var properties []models.Property
	var err error

	if gormDB != nil {
		properties, err = gormDB.GetAllProperties()
	} else {
		properties, err = db.GetAllProperties()
	}

	if err != nil {
		log.Printf("[Reindex] Error fetching properties from database: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch properties from database",
		})
		return
	}

	log.Printf("[Reindex] Found %d properties in database", len(properties))

	// Index all properties to Meilisearch
	successCount := 0
	failCount := 0

	for i, property := range properties {
		if err := searchClient.IndexProperty(&property); err != nil {
			log.Printf("[Reindex] Error indexing property %d (%s): %v", i+1, property.ID, err)
			failCount++
		} else {
			successCount++
		}

		// Log progress every 100 properties
		if (i+1)%100 == 0 {
			log.Printf("[Reindex] Progress: %d/%d indexed", i+1, len(properties))
		}
	}

	log.Printf("[Reindex] Reindex complete. Success: %d, Failed: %d", successCount, failCount)

	c.JSON(http.StatusOK, gin.H{
		"message":       "Reindex complete",
		"total":         len(properties),
		"indexed":       successCount,
		"failed":        failCount,
	})
}

// normalizeURLForCheck normalizes a URL for existence checking
func normalizeURLForCheck(rawURL string) string {
	// Remove query parameters and trailing slash
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	u.RawQuery = ""
	u.Fragment = ""
	normalized := strings.TrimSuffix(u.String(), "/")
	return normalized
}
