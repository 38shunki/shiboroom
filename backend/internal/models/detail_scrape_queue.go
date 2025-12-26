package models

import (
	"time"
)

// DetailScrapeQueue manages pending detail page scrapes
// This allows us to defer scraping and avoid bursts that trigger WAF blocks
type DetailScrapeQueue struct {
	ID               int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	Source           string     `gorm:"type:varchar(50);not null;index:idx_queue_lookup" json:"source"`
	SourcePropertyID string     `gorm:"type:varchar(255);not null;index:idx_queue_lookup" json:"source_property_id"`
	DetailURL        string     `gorm:"type:text;not null" json:"detail_url"`
	Status           string     `gorm:"type:varchar(20);not null;default:'pending';index:idx_status" json:"status"` // pending, processing, done, failed
	Priority         int        `gorm:"default:0;index:idx_priority" json:"priority"`                                // Higher = process first
	Attempts         int        `gorm:"default:0" json:"attempts"`
	LastError        string     `gorm:"type:text" json:"last_error,omitempty"`
	NextRetryAt      *time.Time `gorm:"index:idx_retry" json:"next_retry_at,omitempty"`
	CreatedAt        time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
}

// TableName specifies the table name for GORM
func (DetailScrapeQueue) TableName() string {
	return "detail_scrape_queue"
}

// Status constants
const (
	QueueStatusPending      = "pending"
	QueueStatusProcessing   = "processing"
	QueueStatusDone         = "done"
	QueueStatusFailed       = "failed"
	QueueStatusPermanentFail = "permanent_fail" // 404 or other non-retryable failures
)

// MaxRetryAttempts before marking as permanently failed
const MaxRetryAttempts = 5

// GetNextRetryDelay calculates exponential backoff for retries
func GetNextRetryDelay(attempts int) time.Duration {
	// 5min, 15min, 1h, 4h, 12h
	delays := []time.Duration{
		5 * time.Minute,
		15 * time.Minute,
		1 * time.Hour,
		4 * time.Hour,
		12 * time.Hour,
	}

	if attempts >= len(delays) {
		return delays[len(delays)-1]
	}
	return delays[attempts]
}
