package ratelimit

import (
	"sync"
	"time"
)

// RateLimiter tracks and enforces request rate limits
type RateLimiter struct {
	requestsPerMinute int
	requestsPerHour   int
	requestsPerDay    int
	enabled           bool

	// Request tracking
	minuteWindow []time.Time
	hourWindow   []time.Time
	dayWindow    []time.Time
	mu           sync.Mutex
}

// NewRateLimiter creates a new rate limiter with the given limits
func NewRateLimiter(requestsPerMinute, requestsPerHour, requestsPerDay int, enabled bool) *RateLimiter {
	return &RateLimiter{
		requestsPerMinute: requestsPerMinute,
		requestsPerHour:   requestsPerHour,
		requestsPerDay:    requestsPerDay,
		enabled:           enabled,
		minuteWindow:      make([]time.Time, 0),
		hourWindow:        make([]time.Time, 0),
		dayWindow:         make([]time.Time, 0),
	}
}

// AllowRequest checks if a request is allowed based on rate limits
// Returns true if allowed, false if rate limit exceeded
func (rl *RateLimiter) AllowRequest() bool {
	if !rl.enabled {
		return true
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Clean up old entries
	rl.cleanup(now)

	// Check limits
	if len(rl.minuteWindow) >= rl.requestsPerMinute {
		return false
	}
	if rl.requestsPerHour > 0 && len(rl.hourWindow) >= rl.requestsPerHour {
		return false
	}
	if rl.requestsPerDay > 0 && len(rl.dayWindow) >= rl.requestsPerDay {
		return false
	}

	// Record the request
	rl.minuteWindow = append(rl.minuteWindow, now)
	rl.hourWindow = append(rl.hourWindow, now)
	rl.dayWindow = append(rl.dayWindow, now)

	return true
}

// cleanup removes expired entries from the time windows
func (rl *RateLimiter) cleanup(now time.Time) {
	// Clean minute window (keep last 60 seconds)
	minuteAgo := now.Add(-1 * time.Minute)
	rl.minuteWindow = filterTimes(rl.minuteWindow, minuteAgo)

	// Clean hour window (keep last 60 minutes)
	hourAgo := now.Add(-1 * time.Hour)
	rl.hourWindow = filterTimes(rl.hourWindow, hourAgo)

	// Clean day window (keep last 24 hours)
	dayAgo := now.Add(-24 * time.Hour)
	rl.dayWindow = filterTimes(rl.dayWindow, dayAgo)
}

// filterTimes keeps only times after the cutoff
func filterTimes(times []time.Time, cutoff time.Time) []time.Time {
	result := make([]time.Time, 0, len(times))
	for _, t := range times {
		if t.After(cutoff) {
			result = append(result, t)
		}
	}
	return result
}

// GetStats returns current rate limiter statistics
func (rl *RateLimiter) GetStats() Stats {
	if !rl.enabled {
		return Stats{Enabled: false}
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	rl.cleanup(now)

	return Stats{
		Enabled:              true,
		RequestsLastMinute:   len(rl.minuteWindow),
		RequestsLastHour:     len(rl.hourWindow),
		RequestsLastDay:      len(rl.dayWindow),
		LimitPerMinute:       rl.requestsPerMinute,
		LimitPerHour:         rl.requestsPerHour,
		LimitPerDay:          rl.requestsPerDay,
		RemainingThisMinute:  max(0, rl.requestsPerMinute-len(rl.minuteWindow)),
		RemainingThisHour:    max(0, rl.requestsPerHour-len(rl.hourWindow)),
		RemainingThisDay:     max(0, rl.requestsPerDay-len(rl.dayWindow)),
	}
}

// Stats contains rate limiter statistics
type Stats struct {
	Enabled              bool `json:"enabled"`
	RequestsLastMinute   int  `json:"requests_last_minute"`
	RequestsLastHour     int  `json:"requests_last_hour"`
	RequestsLastDay      int  `json:"requests_last_day"`
	LimitPerMinute       int  `json:"limit_per_minute"`
	LimitPerHour         int  `json:"limit_per_hour"`
	LimitPerDay          int  `json:"limit_per_day"`
	RemainingThisMinute  int  `json:"remaining_this_minute"`
	RemainingThisHour    int  `json:"remaining_this_hour"`
	RemainingThisDay     int  `json:"remaining_this_day"`
}

// Reset clears all tracked requests (useful for testing)
func (rl *RateLimiter) Reset() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.minuteWindow = make([]time.Time, 0)
	rl.hourWindow = make([]time.Time, 0)
	rl.dayWindow = make([]time.Time, 0)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
