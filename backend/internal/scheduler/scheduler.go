package scheduler

import (
	"fmt"
	"log"
	"real-estate-portal/internal/config"
	"real-estate-portal/internal/models"
	"real-estate-portal/internal/snapshot"
	"time"

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
// NOTE: This ONLY enqueues URLs for processing. Actual scraping happens via queue workers.
func (s *Scheduler) runDailyScraping() error {
	// Get all active properties to re-scrape
	var properties []models.Property
	if err := s.db.Where("status = ?", models.PropertyStatusActive).Find(&properties).Error; err != nil {
		return err
	}

	log.Printf("Scheduler: Found %d active properties to enqueue for update", len(properties))

	// Limit: Don't overwhelm the queue (max 100 per scheduler run)
	maxEnqueue := 100
	if len(properties) > maxEnqueue {
		log.Printf("Scheduler: Limiting to %d properties (total: %d)", maxEnqueue, len(properties))
		properties = properties[:maxEnqueue]
	}

	enqueuedCount := 0
	skippedExisting := 0
	skippedDone := 0
	errorCount := 0

	// Enqueue each property URL (no direct scraping!)
	for i, prop := range properties {
		// Extract source_property_id from the property
		// For Yahoo: it's stored in SourcePropertyID field
		if prop.Source == "" || prop.SourcePropertyID == "" || prop.DetailURL == "" {
			log.Printf("Scheduler: [%d/%d] Skipping property %s (missing source/URL)", i+1, len(properties), prop.ID)
			errorCount++
			continue
		}

		// Check if already in queue with pending/processing status
		var existingQueue models.DetailScrapeQueue
		result := s.db.Where("source = ? AND source_property_id = ? AND status IN ?",
			prop.Source, prop.SourcePropertyID, []string{models.QueueStatusPending, models.QueueStatusProcessing}).
			First(&existingQueue)

		if result.Error == nil {
			// Already in queue, skip
			skippedExisting++
			continue
		}

		// Check if recently completed (within 12 hours) to avoid re-scraping too soon
		var recentDone models.DetailScrapeQueue
		twelveHoursAgo := time.Now().Add(-12 * time.Hour)
		resultDone := s.db.Where("source = ? AND source_property_id = ? AND status = ? AND updated_at > ?",
			prop.Source, prop.SourcePropertyID, models.QueueStatusDone, twelveHoursAgo).
			First(&recentDone)

		if resultDone.Error == nil {
			// Recently completed, skip
			skippedDone++
			continue
		}

		// Enqueue for processing
		queue := models.DetailScrapeQueue{
			Source:           prop.Source,
			SourcePropertyID: prop.SourcePropertyID,
			DetailURL:        prop.DetailURL,
			Status:           models.QueueStatusPending,
			Priority:         1, // Scheduled updates have priority 1 (manual can be higher)
		}

		if err := s.db.Create(&queue).Error; err != nil {
			log.Printf("Scheduler: [%d/%d] Failed to enqueue property %s: %v", i+1, len(properties), prop.ID, err)
			errorCount++
			continue
		}

		enqueuedCount++

		if (i+1)%50 == 0 {
			log.Printf("Scheduler: Progress: %d/%d processed", i+1, len(properties))
		}
	}

	log.Printf("Scheduler: Daily enqueue completed. Enqueued=%d, SkippedExisting=%d, SkippedDone=%d, Errors=%d",
		enqueuedCount, skippedExisting, skippedDone, errorCount)

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
