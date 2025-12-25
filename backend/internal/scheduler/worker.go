package scheduler

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"real-estate-portal/internal/models"
	"real-estate-portal/internal/scraper"
	"real-estate-portal/internal/snapshot"
	"strings"
	"time"

	"gorm.io/gorm"
)

// QueueWorker processes detail_scrape_queue items with rate limiting and WAF protection
type QueueWorker struct {
	db                *gorm.DB
	scraper           *scraper.Scraper
	snapshot          *snapshot.Service
	stopChan          chan struct{}
	isRunning         bool
	pollInterval      time.Duration
	maxConcurrency    int
	consecutiveSuccess int // Track consecutive successes for preventive cooldown
}

// NewQueueWorker creates a new queue worker
func NewQueueWorker(db *gorm.DB) *QueueWorker {
	return &QueueWorker{
		db:             db,
		scraper:        scraper.NewScraper(),
		snapshot:       snapshot.NewService(db),
		stopChan:       make(chan struct{}),
		pollInterval:   30 * time.Second, // Check queue every 30 seconds
		maxConcurrency: 1,                // Process 1 at a time (strict rate limiting)
	}
}

// Start starts the queue worker
func (w *QueueWorker) Start() {
	if w.isRunning {
		log.Println("QueueWorker: Already running")
		return
	}

	// WAF Health Check（起動前に1回だけ）
	log.Println("QueueWorker: Running WAF health check...")
	if !w.healthCheck() {
		// WAF detected: enter long cooldown (4 hours minimum)
		log.Println("QueueWorker: WAF detected in health check, entering 4-hour cooldown")
		time.Sleep(4 * time.Hour)

		// Re-check after delay
		if !w.healthCheck() {
			log.Println("QueueWorker: WAF still active after 4h, entering another 4-hour cooldown")
			time.Sleep(4 * time.Hour)

			// Final check
			if !w.healthCheck() {
				log.Println("QueueWorker: WAF persists after 8h total, entering 12-hour cooldown")
				time.Sleep(12 * time.Hour)
			}
		}
	} else {
		log.Println("QueueWorker: Health check passed")
	}

	w.isRunning = true
	log.Printf("QueueWorker: Started (poll_interval=%v, max_concurrency=%d)", w.pollInterval, w.maxConcurrency)

	go w.run()
}

// Stop stops the queue worker
func (w *QueueWorker) Stop() {
	if !w.isRunning {
		return
	}

	log.Println("QueueWorker: Stopping...")
	w.isRunning = false
	close(w.stopChan)
}

// run is the main worker loop
func (w *QueueWorker) run() {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopChan:
			log.Println("QueueWorker: Stopped")
			return
		case <-ticker.C:
			w.processNextBatch()
		}
	}
}

// processNextBatch processes the next batch of queue items
func (w *QueueWorker) processNextBatch() {
	// Get next pending item (ordered by priority desc, then created_at asc)
	var queueItem models.DetailScrapeQueue
	now := time.Now()

	// Priority 1: Try to get a pending item first
	result := w.db.Where("status = ?", models.QueueStatusPending).
		Order("priority DESC, created_at ASC").
		First(&queueItem)

	// Priority 2: If no pending items, try failed items with retry time passed
	if result.Error == gorm.ErrRecordNotFound {
		result = w.db.Where("status = ? AND next_retry_at IS NOT NULL AND next_retry_at <= ?", models.QueueStatusFailed, now).
			Order("priority DESC, created_at ASC").
			First(&queueItem)
	}

	if result.Error != nil {
		if result.Error != gorm.ErrRecordNotFound {
			log.Printf("QueueWorker: Error fetching next queue item: %v", result.Error)
		}
		return
	}

	// Process this item
	w.processQueueItem(&queueItem)
}

// processQueueItem processes a single queue item
func (w *QueueWorker) processQueueItem(item *models.DetailScrapeQueue) {
	log.Printf("QueueWorker: Processing id=%d url=%s attempt=%d", item.ID, item.DetailURL, item.Attempts+1)

	// Mark as processing
	item.Status = models.QueueStatusProcessing
	item.Attempts++
	if err := w.db.Save(item).Error; err != nil {
		log.Printf("QueueWorker: Failed to update status to processing: %v", err)
		return
	}

	// CRITICAL: Apply DetailLimiter (5 per hour max)
	// This is the ONLY place where detail pages should be scraped
	log.Printf("QueueWorker: Acquiring DetailLimiter (caller=worker, id=%d)", item.ID)
	scraper.DetailLimiter.Acquire("worker")

	// Scrape the property
	property, err := w.scraper.ScrapeProperty(item.DetailURL)

	if err != nil {
		w.handleScrapeError(item, err)
		return
	}

	// Get stations from scraper (extracted during scraping)
	stations := w.scraper.GetLastStationsAsModels(property.ID)

	// Success: save property with stations and mark queue item as done
	w.handleScrapeSuccess(item, property, stations)
}

// handleScrapeError handles scraping errors with smart retry logic
func (w *QueueWorker) handleScrapeError(item *models.DetailScrapeQueue, err error) {
	errMsg := err.Error()
	log.Printf("QueueWorker: Scrape failed for id=%d: %v", item.ID, err)

	// Check if it's a permanent failure (404 Not Found)
	if strings.Contains(errMsg, "permanent_fail") || strings.Contains(errMsg, "404") {
		// 404: Property delisted or URL invalid - don't retry
		log.Printf("QueueWorker: Permanent failure (404) for id=%d - marking as permanent_fail (no retry)", item.ID)
		item.Status = models.QueueStatusPermanentFail
		item.LastError = fmt.Sprintf("404 Not Found (permanent): %s", errMsg)
		completedAt := time.Now()
		item.CompletedAt = &completedAt
		item.NextRetryAt = nil

		// Reset consecutive success counter on failure
		w.consecutiveSuccess = 0

		if err := w.db.Save(item).Error; err != nil {
			log.Printf("QueueWorker: Failed to save permanent_fail status: %v", err)
		}
		return
	}

	// Check for WAF block
	if strings.Contains(errMsg, "WAF") || strings.Contains(errMsg, "circuit breaker open") {
		log.Printf("QueueWorker: WAF/circuit breaker detected for id=%d - entering cooldown", item.ID)

		// Reset consecutive success counter on WAF
		w.consecutiveSuccess = 0

		// WAF detected: enter long cooldown (1 hour minimum)
		item.Status = models.QueueStatusFailed
		item.LastError = fmt.Sprintf("WAF/circuit breaker: %s", errMsg)
		nextRetry := time.Now().Add(1 * time.Hour)
		item.NextRetryAt = &nextRetry

		if err := w.db.Save(item).Error; err != nil {
			log.Printf("QueueWorker: Failed to save WAF cooldown: %v", err)
		}

		// Also: pause worker for a bit to let circuit breaker reset
		log.Printf("QueueWorker: Pausing for 5 minutes due to WAF detection")
		time.Sleep(5 * time.Minute)
		return
	}

	// Retryable error (500, 503, timeout, etc.)
	// Reset consecutive success counter on any error
	w.consecutiveSuccess = 0

	if item.Attempts >= models.MaxRetryAttempts {
		// Max retries exceeded
		log.Printf("QueueWorker: Max retries exceeded for id=%d (%d attempts)", item.ID, item.Attempts)
		item.Status = models.QueueStatusFailed
		item.LastError = fmt.Sprintf("Max retries exceeded (%d): %s", item.Attempts, errMsg)
		completedAt := time.Now()
		item.CompletedAt = &completedAt
		item.NextRetryAt = nil
	} else {
		// Schedule retry with exponential backoff
		delay := models.GetNextRetryDelay(item.Attempts - 1) // -1 because we already incremented Attempts
		nextRetry := time.Now().Add(delay)
		item.Status = models.QueueStatusFailed
		item.LastError = errMsg
		item.NextRetryAt = &nextRetry
		log.Printf("QueueWorker: Scheduling retry for id=%d in %v (attempt %d/%d)",
			item.ID, delay, item.Attempts, models.MaxRetryAttempts)
	}

	if err := w.db.Save(item).Error; err != nil {
		log.Printf("QueueWorker: Failed to save retry status: %v", err)
	}
}

// handleScrapeSuccess handles successful scraping
func (w *QueueWorker) handleScrapeSuccess(item *models.DetailScrapeQueue, property *models.Property, stations []models.PropertyStation) {
	log.Printf("QueueWorker: Successfully scraped id=%d property_id=%s stations=%d", item.ID, property.ID, len(stations))

	// Check if property already exists
	var existing models.Property
	result := w.db.Where("source = ? AND source_property_id = ?", property.Source, property.SourcePropertyID).
		First(&existing)

	if result.Error == nil {
		// Property exists: preserve ID and created_at
		property.ID = existing.ID
		property.CreatedAt = existing.CreatedAt

		// Detect changes for snapshot
		changes, err := w.snapshot.DetectChanges(property)
		if err != nil {
			log.Printf("QueueWorker: Failed to detect changes: %v", err)
		} else if len(changes) > 0 {
			log.Printf("QueueWorker: Detected %d changes for property %s", len(changes), property.ID)
		}
	}

	// Save property with stations to database (transaction-based)
	// Create GormDB wrapper from the worker's db instance
	gormDB := database.NewGormDBFromDB(w.db)
	if err := gormDB.SavePropertyWithStations(property, stations); err != nil {
		log.Printf("QueueWorker: Failed to save property with stations: %v", err)
		// Treat as retryable error
		w.handleScrapeError(item, fmt.Errorf("database save error: %w", err))
		return
	}

	if len(stations) == 0 {
		log.Printf("QueueWorker: [stations] property_id=%s stations_len=0 skip_delete_preserve_existing", property.ID)
	} else {
		log.Printf("QueueWorker: [stations] property_id=%s stations_len=%d saved", property.ID, len(stations))
	}

	// Create snapshot with change detection
	if err := w.snapshot.CreateSnapshotWithChangeDetection(property); err != nil {
		log.Printf("QueueWorker: Warning: Failed to create snapshot: %v", err)
		// Don't fail the whole operation for snapshot errors
	}

	// Mark queue item as done
	item.Status = models.QueueStatusDone
	item.LastError = ""
	completedAt := time.Now()
	item.CompletedAt = &completedAt
	item.NextRetryAt = nil

	if err := w.db.Save(item).Error; err != nil {
		log.Printf("QueueWorker: Failed to mark item as done: %v", err)
	} else {
		log.Printf("QueueWorker: ✅ Completed id=%d property_id=%s", item.ID, property.ID)

		// Track consecutive successes for preventive cooldown
		w.consecutiveSuccess++

		// Preventive cooldown after 3 consecutive successes (simulate human behavior)
		if w.consecutiveSuccess >= 3 {
			cooldownDuration := 5 * time.Minute
			log.Printf("QueueWorker: Preventive cooldown after %d successes - pausing for %v", w.consecutiveSuccess, cooldownDuration)
			time.Sleep(cooldownDuration)
			w.consecutiveSuccess = 0 // Reset counter
		}
	}
}

// healthCheck performs a lightweight request to check for WAF blocks
func (w *QueueWorker) healthCheck() bool {
	testURL := "https://realestate.yahoo.co.jp/rent/"
	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		log.Printf("QueueWorker: Health check request creation failed: %v", err)
		return false
	}

	// Apply browser-like headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "ja-JP,ja;q=0.9,en-US;q=0.8,en;q=0.7")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("QueueWorker: Health check network error: %v", err)
		return false
	}
	defer resp.Body.Close()

	// Check for WAF block
	if resp.StatusCode >= 500 {
		body, _ := io.ReadAll(resp.Body)
		if strings.Contains(string(body), "ご覧になろうとしているページは現在表示できません") {
			log.Printf("QueueWorker: WAF block detected in health check (status: %d)", resp.StatusCode)
			return false
		}
	}

	// 403 also could be WAF
	if resp.StatusCode == 403 {
		log.Printf("QueueWorker: 403 Forbidden in health check - possible WAF")
		return false
	}

	log.Printf("QueueWorker: Health check OK (status: %d)", resp.StatusCode)
	return true
}

// GetQueueStats returns current queue statistics
func (w *QueueWorker) GetQueueStats() map[string]interface{} {
	var stats struct {
		Pending       int64
		Processing    int64
		Done          int64
		Failed        int64
		PermanentFail int64
	}

	w.db.Model(&models.DetailScrapeQueue{}).Where("status = ?", models.QueueStatusPending).Count(&stats.Pending)
	w.db.Model(&models.DetailScrapeQueue{}).Where("status = ?", models.QueueStatusProcessing).Count(&stats.Processing)
	w.db.Model(&models.DetailScrapeQueue{}).Where("status = ?", models.QueueStatusDone).Count(&stats.Done)
	w.db.Model(&models.DetailScrapeQueue{}).Where("status = ?", models.QueueStatusFailed).Count(&stats.Failed)
	w.db.Model(&models.DetailScrapeQueue{}).Where("status = ?", models.QueueStatusPermanentFail).Count(&stats.PermanentFail)

	return map[string]interface{}{
		"pending":        stats.Pending,
		"processing":     stats.Processing,
		"done":           stats.Done,
		"failed":         stats.Failed,
		"permanent_fail": stats.PermanentFail,
		"is_running":     w.isRunning,
	}
}
