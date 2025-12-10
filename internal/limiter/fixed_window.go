package limiter

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// FixedWindowLimiter implements the fixed window algorithm
// Resets counter at fixed time intervals
type FixedWindowLimiter struct {
	mu           sync.Mutex
	limit        Limit
	maxCount     int
	window       time.Duration
	currentCount int
	windowStart  time.Time
}

// NewFixedWindow creates a new fixed window limiter
func NewFixedWindow(r Limit, maxCount int) *FixedWindowLimiter {
	window := time.Second
	if r > 0 {
		window = time.Duration(float64(time.Second) * float64(maxCount) / float64(r))
	}

	return &FixedWindowLimiter{
		limit:        r,
		maxCount:     maxCount,
		window:       window,
		currentCount: 0,
		windowStart:  time.Now(),
	}
}

func (fw *FixedWindowLimiter) Algorithm() Algorithm {
	return FixedWindow
}

func (fw *FixedWindowLimiter) Capabilities() Capabilities {
	return Capabilities{
		SupportsTokens:      false,
		SupportsBurst:       false,
		SupportsReservation: false,
	}
}

// resetIfNeeded resets the counter if we're in a new window
func (fw *FixedWindowLimiter) resetIfNeeded(now time.Time) {
	if now.Sub(fw.windowStart) >= fw.window {
		fw.currentCount = 0
		fw.windowStart = now.Truncate(fw.window)
	}
}

func (fw *FixedWindowLimiter) Allow() bool {
	return fw.AllowN(time.Now(), 1)
}

func (fw *FixedWindowLimiter) AllowN(t time.Time, n int) bool {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	fw.resetIfNeeded(t)

	if fw.currentCount+n <= fw.maxCount {
		fw.currentCount += n
		return true
	}
	return false
}

func (fw *FixedWindowLimiter) Reserve() *Reservation {
	return fw.ReserveN(time.Now(), 1)
}

func (fw *FixedWindowLimiter) ReserveN(t time.Time, n int) *Reservation {
	if fw.AllowN(t, n) {
		return &Reservation{
			ok:        true,
			lim:       fw,
			tokens:    n,
			timeToAct: t,
			limit:     fw.limit,
		}
	}
	return &Reservation{ok: false}
}

func (fw *FixedWindowLimiter) Wait(ctx context.Context) error {
	return fw.WaitN(ctx, 1)
}

func (fw *FixedWindowLimiter) WaitN(ctx context.Context, n int) error {
	fw.mu.Lock()
	now := time.Now()
	fw.resetIfNeeded(now)

	if n > fw.maxCount {
		fw.mu.Unlock()
		return fmt.Errorf("rate: requested tokens (%d) exceeds limit (%d)", n, fw.maxCount)
	}

	if fw.currentCount+n > fw.maxCount {
		// Wait for next window
		nextWindow := fw.windowStart.Add(fw.window)
		fw.mu.Unlock()

		select {
		case <-time.After(time.Until(nextWindow)):
			return fw.WaitN(ctx, n) // Retry in new window
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	fw.currentCount += n
	fw.mu.Unlock()
	return nil
}

func (fw *FixedWindowLimiter) Limit() Limit {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	return fw.limit
}

func (fw *FixedWindowLimiter) SetLimit(newLimit Limit) {
	fw.SetLimitAt(time.Now(), newLimit)
}

func (fw *FixedWindowLimiter) SetLimitAt(t time.Time, newLimit Limit) {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	fw.resetIfNeeded(t)
	fw.limit = newLimit
	if newLimit > 0 {
		fw.window = time.Duration(float64(time.Second) * float64(fw.maxCount) / float64(newLimit))
	}
}

func (fw *FixedWindowLimiter) Burst() int {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	return fw.maxCount
}

func (fw *FixedWindowLimiter) SetBurst(newBurst int) {
	fw.SetBurstAt(time.Now(), newBurst)
}

func (fw *FixedWindowLimiter) SetBurstAt(t time.Time, newBurst int) {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	fw.resetIfNeeded(t)
	fw.maxCount = newBurst
	if fw.limit > 0 {
		fw.window = time.Duration(float64(time.Second) * float64(newBurst) / float64(fw.limit))
	}
}

// Tokens returns remaining capacity in current window
func (fw *FixedWindowLimiter) Tokens() float64 {
	return fw.TokensAt(time.Now())
}

func (fw *FixedWindowLimiter) TokensAt(t time.Time) float64 {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	fw.resetIfNeeded(t)
	return float64(fw.maxCount - fw.currentCount)
}
