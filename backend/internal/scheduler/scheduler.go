package scheduler

import (
	"fmt"
	"log"
	"real-estate-portal/internal/config"
	"real-estate-portal/internal/models"
	"real-estate-portal/internal/scraper"
	"real-estate-portal/internal/snapshot"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

// Scheduler handles scheduled scraping tasks
type Scheduler struct {
	cron      *cron.Cron
	db        *gorm.DB
	snapshot  *snapshot.Service
	config    *config.Config
	isRunning bool
}

// NewScheduler creates a new scheduler
func NewScheduler(db *gorm.DB, cfg *config.Config) *Scheduler {
	return &Scheduler{
		cron:     cron.New(),
		db:       db,
		snapshot: snapshot.NewService(db),
		config:   cfg,
	}
}

// Start starts the scheduler
func (s *Scheduler) Start() error {
	if !s.config.Scraper.DailyRunEnabled {
		log.Println("Scheduler: Daily run is disabled in configuration")
		return nil
	}

	// Parse daily run time (HH:MM format in config)
	cronSpec := s.parseDailyRunTime(s.config.Scraper.DailyRunTime)

	// Add daily scraping job
	_, err := s.cron.AddFunc(cronSpec, func() {
		log.Println("Scheduler: Starting daily scraping job...")
		if err := s.runDailyScraping(); err != nil {
			log.Printf("Scheduler: Daily scraping failed: %v", err)
		} else {
			log.Println("Scheduler: Daily scraping completed successfully")
		}
	})

	if err != nil {
		return err
	}

	s.cron.Start()
	s.isRunning = true
	log.Printf("Scheduler: Started with daily run at %s (cron: %s)", s.config.Scraper.DailyRunTime, cronSpec)

	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	if s.isRunning {
		s.cron.Stop()
		s.isRunning = false
		log.Println("Scheduler: Stopped")
	}
}

// runDailyScraping executes the daily scraping routine
func (s *Scheduler) runDailyScraping() error {
	// Get all active properties to re-scrape
	var properties []models.Property
	if err := s.db.Where("status = ?", models.PropertyStatusActive).Find(&properties).Error; err != nil {
		return err
	}

	log.Printf("Scheduler: Found %d active properties to update", len(properties))

	// Create scraper with configuration
	sc := scraper.NewScraperWithConfig(scraper.ScraperConfig{
		Timeout:      s.config.Scraper.GetTimeout(),
		MaxRetries:   s.config.Scraper.MaxRetries,
		RetryDelay:   s.config.Scraper.GetRetryDelay(),
		RequestDelay: s.config.Scraper.GetRequestDelay(),
	})

	successCount := 0
	errorCount := 0
	changedCount := 0

	// Re-scrape each property
	for i, prop := range properties {
		log.Printf("Scheduler: [%d/%d] Updating property %s", i+1, len(properties), prop.ID)

		// Scrape the property page
		updatedProp, err := sc.ScrapeProperty(prop.DetailURL)
		if err != nil {
			log.Printf("Scheduler: Failed to scrape property %s: %v", prop.ID, err)
			errorCount++

			// If property is no longer accessible, mark as removed
			if s.config.Scraper.StopOnError {
				log.Println("Scheduler: Stop on error is enabled, stopping daily scraping")
				break
			}
			continue
		}

		// Preserve original ID and created_at
		updatedProp.ID = prop.ID
		updatedProp.CreatedAt = prop.CreatedAt

		// Detect changes before updating
		changes, err := s.snapshot.DetectChanges(updatedProp)
		if err != nil {
			log.Printf("Scheduler: Failed to detect changes for property %s: %v", prop.ID, err)
		}

		if len(changes) > 0 {
			changedCount++
			log.Printf("Scheduler: Property %s has %d changes", prop.ID, len(changes))
		}

		// Update property in database
		if err := s.db.Save(updatedProp).Error; err != nil {
			log.Printf("Scheduler: Failed to save property %s: %v", prop.ID, err)
			errorCount++
			continue
		}

		// Create snapshot with change detection
		if err := s.snapshot.CreateSnapshotWithChangeDetection(updatedProp); err != nil {
			log.Printf("Scheduler: Failed to create snapshot for property %s: %v", prop.ID, err)
		}

		successCount++
	}

	log.Printf("Scheduler: Daily scraping completed. Success: %d, Errors: %d, Changed: %d",
		successCount, errorCount, changedCount)

	return nil
}

// RunNow immediately executes the daily scraping job (for manual trigger)
func (s *Scheduler) RunNow() error {
	log.Println("Scheduler: Manual trigger - starting scraping job...")
	return s.runDailyScraping()
}

// parseDailyRunTime converts HH:MM format to cron specification
// Example: "02:00" -> "0 2 * * *" (run at 2:00 AM every day)
func (s *Scheduler) parseDailyRunTime(timeStr string) string {
	// timeStr is expected to be in "HH:MM" format
	// Convert to cron format: "minute hour * * *"
	var hour, minute int
	n, _ := fmt.Sscanf(timeStr, "%d:%d", &hour, &minute)
	if n == 2 {
		return fmt.Sprintf("%d %d * * *", minute, hour)
	}

	// Default to 2:00 AM if parsing fails
	log.Printf("Scheduler: Failed to parse time '%s', using default 02:00", timeStr)
	return "0 2 * * *"
}
