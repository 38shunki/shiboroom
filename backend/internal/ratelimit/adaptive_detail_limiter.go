package ratelimit

import (
	"log"
	"math"
	"sync"
	"time"
)

type DetailRateConfig struct {
	NightPerHour   int
	DayPerHour     int
	DefaultPerHour int
	NightStart     int // 2
	NightEnd       int // 6
	DayStart       int // 10
	DayEnd         int // 22
}

type AdaptiveConfig struct {
	Window           int
	SlowThreshold    float64 // 0.20
	RecoverThreshold float64 // 0.10
	SlowPerHour      int     // 5

	Cooldown        time.Duration // 60m
	RampStep        int           // 2
	RampMinInterval time.Duration // 30m
}

type AdaptiveDetailLimiter struct {
	mu sync.Mutex

	base DetailRateConfig
	ada  AdaptiveConfig

	// existing limiter cache (perHour -> limiter)
	limiters map[int]*DetailLimiter

	// sliding window (true=success)
	results []bool
	idx     int
	filled  bool

	// state machine
	slowUntil       time.Time
	currentCapPerHr int
	nextRampAt      time.Time

	// pacing: enforce minimum interval on top (prevents mid-hour limiter switch loophole)
	lastAcquireAt time.Time
}

func NewAdaptiveDetailLimiter(base DetailRateConfig, ada AdaptiveConfig) *AdaptiveDetailLimiter {
	// sane defaults
	if ada.Window <= 0 {
		ada.Window = 20
	}
	if ada.SlowThreshold <= 0 {
		ada.SlowThreshold = 0.20
	}
	if ada.RecoverThreshold <= 0 {
		ada.RecoverThreshold = 0.10
	}
	if ada.SlowPerHour <= 0 {
		ada.SlowPerHour = 5
	}
	if ada.Cooldown <= 0 {
		ada.Cooldown = 60 * time.Minute
	}
	if ada.RampStep <= 0 {
		ada.RampStep = 2
	}
	if ada.RampMinInterval <= 0 {
		ada.RampMinInterval = 30 * time.Minute
	}

	return &AdaptiveDetailLimiter{
		base:     base,
		ada:      ada,
		limiters: make(map[int]*DetailLimiter),
		results:  make([]bool, ada.Window),
	}
}

// Acquire keeps the same signature as DetailLimiter.Acquire(caller)
func (l *AdaptiveDetailLimiter) Acquire(caller string) {
	perHr, failRate, slow, capPerHr, sleep := l.prepare(caller)

	// 1) pacing layer (global) - prevents exceeding when perHr changes mid-hour
	if sleep > 0 {
		time.Sleep(sleep)
	}

	// 2) existing limiter as a backstop (keeps existing behavior/logs)
	lim := l.getOrCreateLimiter(perHr)
	lim.Acquire(caller)

	// mark lastAcquireAt after limiter passes
	l.mu.Lock()
	l.lastAcquireAt = time.Now()
	l.mu.Unlock()

	// optional: adaptive debug log (keep prefix compatible)
	log.Printf("[DetailLimiter] caller=%s mode=adaptive perHr=%d failRate=%.2f slow=%t cap=%d",
		caller, perHr, failRate, slow, capPerHr)
}

// Observe should be called once per detail attempt (success=true/false)
func (l *AdaptiveDetailLimiter) Observe(success bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// record in ring
	l.results[l.idx] = success
	l.idx++
	if l.idx >= len(l.results) {
		l.idx = 0
		l.filled = true
	}

	failRate := l.failureRateLocked()
	now := time.Now()

	// enter slow mode
	if failRate >= l.ada.SlowThreshold {
		l.slowUntil = now.Add(l.ada.Cooldown)
		l.currentCapPerHr = l.ada.SlowPerHour
		l.nextRampAt = l.slowUntil.Add(l.ada.RampMinInterval)
		log.Printf("[DetailLimiter] ⚠️  Entering slow mode: failRate=%.2f threshold=%.2f cooldown=%v",
			failRate, l.ada.SlowThreshold, l.ada.Cooldown)
		return
	}

	// during cooldown do nothing
	if now.Before(l.slowUntil) {
		return
	}

	// ramp-up only when stable enough
	if failRate <= l.ada.RecoverThreshold {
		if now.After(l.nextRampAt) {
			// if cap not set, start from slow cap
			if l.currentCapPerHr <= 0 {
				l.currentCapPerHr = l.ada.SlowPerHour
			}
			oldCap := l.currentCapPerHr
			l.currentCapPerHr += l.ada.RampStep
			l.nextRampAt = now.Add(l.ada.RampMinInterval)
			log.Printf("[DetailLimiter] ✅ Ramping up: %d -> %d/hr (failRate=%.2f)",
				oldCap, l.currentCapPerHr, failRate)
		}
	}
}

func (l *AdaptiveDetailLimiter) prepare(caller string) (perHr int, failRate float64, slow bool, capPerHr int, sleep time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()

	base := l.basePerHourLocked(now)
	failRate = l.failureRateLocked()

	// state & effective cap
	slow = now.Before(l.slowUntil)
	capPerHr = 0

	if slow {
		capPerHr = minInt(base, l.ada.SlowPerHour)
	} else if l.currentCapPerHr > 0 {
		capPerHr = minInt(base, l.currentCapPerHr)
	} else {
		capPerHr = base
	}

	// clamp
	perHr = clampInt(capPerHr, 1, 60) // 1〜60/h（上限は安全側に適当）
	// pacing interval
	interval := time.Duration(math.Round(float64(time.Hour) / float64(perHr)))

	// compute sleep required
	if !l.lastAcquireAt.IsZero() {
		nextAllowed := l.lastAcquireAt.Add(interval)
		if now.Before(nextAllowed) {
			sleep = nextAllowed.Sub(now)
		}
	}

	return
}

func (l *AdaptiveDetailLimiter) getOrCreateLimiter(perHr int) *DetailLimiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	perHr = clampInt(perHr, 1, 60)
	if lim, ok := l.limiters[perHr]; ok {
		return lim
	}
	lim := NewDetailLimiter(perHr)
	l.limiters[perHr] = lim
	return lim
}

func (l *AdaptiveDetailLimiter) basePerHourLocked(now time.Time) int {
	h := now.Hour()

	// night
	if inHourRange(h, l.base.NightStart, l.base.NightEnd) {
		return clampInt(l.base.NightPerHour, 1, 60)
	}
	// day
	if inHourRange(h, l.base.DayStart, l.base.DayEnd) {
		return clampInt(l.base.DayPerHour, 1, 60)
	}
	return clampInt(l.base.DefaultPerHour, 1, 60)
}

func inHourRange(h, start, end int) bool {
	// [start, end) / supports wrap-around
	start = ((start % 24) + 24) % 24
	end = ((end % 24) + 24) % 24
	h = ((h % 24) + 24) % 24

	if start < end {
		return h >= start && h < end
	}
	// wrap
	return h >= start || h < end
}

func (l *AdaptiveDetailLimiter) failureRateLocked() float64 {
	n := len(l.results)
	if !l.filled {
		n = l.idx
	}
	if n <= 0 {
		return 0
	}
	fail := 0
	for i := 0; i < n; i++ {
		if !l.results[i] {
			fail++
		}
	}
	return float64(fail) / float64(n)
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
