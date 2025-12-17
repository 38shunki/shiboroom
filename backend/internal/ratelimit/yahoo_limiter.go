package ratelimit

import (
	"math/rand"
	"sync"
	"time"
)

// YahooLimiter manages rate limiting for Yahoo Real Estate scraping
type YahooLimiter struct {
	maxInFlight    int           // Maximum concurrent HTTP requests to Yahoo
	currentInFlight int
	mutex          sync.Mutex
	baseDelay      time.Duration // Base delay between requests
	jitter         time.Duration // Random jitter to add
	lastRequest    time.Time
}

// NewYahooLimiter creates a new rate limiter for Yahoo scraping
func NewYahooLimiter(maxInFlight int, baseDelay, jitter time.Duration) *YahooLimiter {
	return &YahooLimiter{
		maxInFlight: maxInFlight,
		baseDelay:   baseDelay,
		jitter:      jitter,
		lastRequest: time.Now(),
	}
}

// Acquire waits until it's safe to make a request
func (yl *YahooLimiter) Acquire() {
	yl.mutex.Lock()

	// Wait for in-flight count to drop
	for yl.currentInFlight >= yl.maxInFlight {
		yl.mutex.Unlock()
		time.Sleep(100 * time.Millisecond)
		yl.mutex.Lock()
	}

	// Apply rate limiting with jitter
	elapsed := time.Since(yl.lastRequest)
	requiredDelay := yl.baseDelay + time.Duration(rand.Int63n(int64(yl.jitter)))

	if elapsed < requiredDelay {
		time.Sleep(requiredDelay - elapsed)
	}

	yl.currentInFlight++
	yl.lastRequest = time.Now()
	yl.mutex.Unlock()
}

// Release marks a request as completed
func (yl *YahooLimiter) Release() {
	yl.mutex.Lock()
	yl.currentInFlight--
	yl.mutex.Unlock()
}

// GetInFlight returns current in-flight request count (for debugging)
func (yl *YahooLimiter) GetInFlight() int {
	yl.mutex.Lock()
	defer yl.mutex.Unlock()
	return yl.currentInFlight
}
