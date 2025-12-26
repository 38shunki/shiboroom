package ratelimit

import (
	"log"
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

// DetailLimiter manages very slow rate limiting for detail pages (5 per hour)
type DetailLimiter struct {
	mutex         sync.Mutex
	requestTimes  []time.Time
	maxPerHour    int
	windowDuration time.Duration
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

// NewDetailLimiter creates a new detail page rate limiter
func NewDetailLimiter(maxPerHour int) *DetailLimiter {
	return &DetailLimiter{
		requestTimes:   make([]time.Time, 0),
		maxPerHour:     maxPerHour,
		windowDuration: 1 * time.Hour,
	}
}

// Acquire waits until it's safe to make a detail page request
func (dl *DetailLimiter) Acquire(caller string) {
	dl.mutex.Lock()
	defer dl.mutex.Unlock()

	now := time.Now()
	nowEpoch := now.Unix()
	windowStart := now.Add(-dl.windowDuration)

	// Remove old requests outside the window
	validRequests := make([]time.Time, 0)
	for _, t := range dl.requestTimes {
		if t.After(windowStart) {
			validRequests = append(validRequests, t)
		}
	}
	dl.requestTimes = validRequests

	// If we've hit the limit, wait until the oldest request expires
	for len(dl.requestTimes) >= dl.maxPerHour {
		oldestRequest := dl.requestTimes[0]
		waitUntil := oldestRequest.Add(dl.windowDuration)
		nextEpoch := waitUntil.Unix()
		waitDuration := time.Until(waitUntil)
		waitSec := int(waitDuration.Seconds())

		if waitDuration > 0 {
			log.Printf("[DetailLimiter] caller=%s limiter=detail now_epoch=%d next_epoch=%d wait_sec=%d reason=rate_limit count=%d/%d",
				caller, nowEpoch, nextEpoch, waitSec, len(dl.requestTimes), dl.maxPerHour)
			dl.mutex.Unlock()
			time.Sleep(waitDuration + 1*time.Second)
			dl.mutex.Lock()

			// Re-check after waiting
			now = time.Now()
			nowEpoch = now.Unix()
			windowStart = now.Add(-dl.windowDuration)
			validRequests = make([]time.Time, 0)
			for _, t := range dl.requestTimes {
				if t.After(windowStart) {
					validRequests = append(validRequests, t)
				}
			}
			dl.requestTimes = validRequests
		} else {
			break
		}
	}

	// Record this request
	dl.requestTimes = append(dl.requestTimes, now)
	log.Printf("[DetailLimiter] caller=%s Request allowed (%d/%d used in last hour)",
		caller, len(dl.requestTimes), dl.maxPerHour)
}

// GetUsage returns current usage count in the window
func (dl *DetailLimiter) GetUsage() int {
	dl.mutex.Lock()
	defer dl.mutex.Unlock()

	now := time.Now()
	windowStart := now.Add(-dl.windowDuration)

	count := 0
	for _, t := range dl.requestTimes {
		if t.After(windowStart) {
			count++
		}
	}
	return count
}
