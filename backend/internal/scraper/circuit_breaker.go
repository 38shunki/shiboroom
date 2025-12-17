package scraper

import (
	"log"
	"sync"
	"time"
)

// CircuitBreaker prevents continued scraping when blocked
type CircuitBreaker struct {
	failureThreshold int
	resetTimeout     time.Duration

	failures           int
	successes          int
	totalRequests      int
	consecutiveFailures int  // NEW: Track consecutive failures for immediate detection
	isOpen             bool
	lastFailureTime    time.Time

	mutex              sync.Mutex
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(failureThreshold int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		failureThreshold: failureThreshold,
		resetTimeout:     resetTimeout,
	}
}

// RecordSuccess records a successful request
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.successes++
	cb.totalRequests++

	// Reset consecutive failures on success
	cb.consecutiveFailures = 0

	// Keep cumulative failures for rate calculation
	// (don't reset cb.failures here - we need it for the 20-request window)
}

// RecordFailure records a failed request (500, 503, etc.)
func (cb *CircuitBreaker) RecordFailure(statusCode int) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.failures++
	cb.consecutiveFailures++
	cb.totalRequests++
	cb.lastFailureTime = time.Now()

	// IMMEDIATE STOP: 2 consecutive critical errors = instant block detected
	if cb.consecutiveFailures >= 2 && (statusCode == 500 || statusCode == 429 || statusCode == 403) {
		cb.isOpen = true
		log.Printf("🚨 CIRCUIT BREAKER OPEN: %d consecutive %d errors. WAF block detected!", cb.consecutiveFailures, statusCode)
		log.Printf("⚠️  Scraping halted immediately. Will retry after %v", cb.resetTimeout)
		return
	}

	// GRADUAL DETECTION: Check failure rate after 20 requests (insurance)
	if cb.totalRequests >= 20 {
		failureRate := float64(cb.failures) / float64(cb.totalRequests)

		if failureRate >= 0.40 { // 40% failure rate (stricter than before)
			cb.isOpen = true
			log.Printf("⚠️  CIRCUIT BREAKER OPEN: Failure rate %.1f%% (%d/%d failures). Suspected WAF block.",
				failureRate*100, cb.failures, cb.totalRequests)
			log.Printf("⚠️  Scraping halted. Will retry after %v", cb.resetTimeout)
		}
	}
}

// CanProceed checks if requests are allowed
func (cb *CircuitBreaker) CanProceed() bool {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	if !cb.isOpen {
		return true
	}

	// Check if reset timeout has passed
	if time.Since(cb.lastFailureTime) > cb.resetTimeout {
		log.Printf("Circuit breaker attempting half-open state after %v", cb.resetTimeout)
		cb.isOpen = false
		cb.failures = 0
		cb.successes = 0
		cb.totalRequests = 0
		cb.consecutiveFailures = 0
		return true
	}

	return false
}

// GetStatus returns current circuit breaker status
func (cb *CircuitBreaker) GetStatus() (isOpen bool, failures int, total int) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	return cb.isOpen, cb.failures, cb.totalRequests
}
