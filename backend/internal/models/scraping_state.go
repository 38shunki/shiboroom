package models

import "time"

// ScrapingState tracks scraping job status and blocking state
type ScrapingState struct {
	ID            int       `gorm:"primaryKey" json:"id"`
	IsBlocked     bool      `gorm:"not null;default:false" json:"is_blocked"`
	BlockedUntil  *time.Time `gorm:"index" json:"blocked_until,omitempty"`
	BlockedReason string    `json:"blocked_reason,omitempty"`
	LastAttempt   time.Time `gorm:"not null" json:"last_attempt"`
	LastSuccess   *time.Time `json:"last_success,omitempty"`
	FailureCount  int       `gorm:"not null;default:0" json:"failure_count"`
	SuccessCount  int       `gorm:"not null;default:0" json:"success_count"`
	CreatedAt     time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt     time.Time `gorm:"not null" json:"updated_at"`
}

// TableName specifies the table name
func (ScrapingState) TableName() string {
	return "scraping_state"
}

// CanScrape checks if scraping is allowed (not blocked)
func (s *ScrapingState) CanScrape() bool {
	if !s.IsBlocked {
		return true
	}

	if s.BlockedUntil == nil {
		return false
	}

	// Check if cooling period has passed
	return time.Now().After(*s.BlockedUntil)
}

// SetBlocked marks scraping as blocked with cooling period
func (s *ScrapingState) SetBlocked(reason string, coolingPeriod time.Duration) {
	s.IsBlocked = true
	s.BlockedReason = reason
	blockedUntil := time.Now().Add(coolingPeriod)
	s.BlockedUntil = &blockedUntil
	s.LastAttempt = time.Now()
}

// ClearBlock clears the blocked state
func (s *ScrapingState) ClearBlock() {
	s.IsBlocked = false
	s.BlockedUntil = nil
	s.BlockedReason = ""
}

// RecordSuccess records a successful scraping attempt
func (s *ScrapingState) RecordSuccess() {
	s.SuccessCount++
	s.FailureCount = 0 // Reset failure count on success
	now := time.Now()
	s.LastSuccess = &now
	s.LastAttempt = now
	s.ClearBlock() // Clear any existing block on success
}

// RecordFailure records a failed scraping attempt
func (s *ScrapingState) RecordFailure() {
	s.FailureCount++
	s.LastAttempt = time.Now()
}
