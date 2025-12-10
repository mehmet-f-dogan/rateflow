package limiter

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// SlidingWindowLimiter implements the sliding window algorithm
// Tracks requests within a rolling time window
type SlidingWindowLimiter struct {
	mu         sync.Mutex
	limit      Limit
	maxCount   int
	window     time.Duration
	timestamps []time.Time
}

// NewSlidingWindow creates a new sliding window limiter
func NewSlidingWindow(r Limit, maxCount int) *SlidingWindowLimiter {
	window := time.Second
	if r > 0 {
		window = time.Duration(float64(time.Second) * float64(maxCount) / float64(r))
	}

	return &SlidingWindowLimiter{
		limit:      r,
		maxCount:   maxCount,
		window:     window,
		timestamps: make([]time.Time, 0, maxCount),
	}
}

func (sw *SlidingWindowLimiter) Algorithm() Algorithm {
	return SlidingWindow
}

func (sw *SlidingWindowLimiter) Capabilities() Capabilities {
	return Capabilities{
		SupportsTokens:      false,
		SupportsBurst:       false,
		SupportsReservation: false,
	}
}

// cleanup removes timestamps outside the current window
func (sw *SlidingWindowLimiter) cleanup(now time.Time) {
	cutoff := now.Add(-sw.window)
	validIdx := 0
	for validIdx < len(sw.timestamps) && sw.timestamps[validIdx].Before(cutoff) {
		validIdx++
	}
	sw.timestamps = sw.timestamps[validIdx:]
}

func (sw *SlidingWindowLimiter) Allow() bool {
	return sw.AllowN(time.Now(), 1)
}

func (sw *SlidingWindowLimiter) AllowN(t time.Time, n int) bool {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	sw.cleanup(t)

	if len(sw.timestamps)+n <= sw.maxCount {
		for i := 0; i < n; i++ {
			sw.timestamps = append(sw.timestamps, t)
		}
		return true
	}
	return false
}

// Reserve returns a reservation that's either immediate or not OK
// (sliding window can't predict future availability)
func (sw *SlidingWindowLimiter) Reserve() *Reservation {
	return sw.ReserveN(time.Now(), 1)
}

func (sw *SlidingWindowLimiter) ReserveN(t time.Time, n int) *Reservation {
	if sw.AllowN(t, n) {
		return &Reservation{
			ok:        true,
			lim:       sw,
			tokens:    n,
			timeToAct: t,
			limit:     sw.limit,
		}
	}
	return &Reservation{ok: false}
}

func (sw *SlidingWindowLimiter) Wait(ctx context.Context) error {
	return sw.WaitN(ctx, 1)
}

func (sw *SlidingWindowLimiter) WaitN(ctx context.Context, n int) error {
	sw.mu.Lock()
	now := time.Now()
	sw.cleanup(now)

	if n > sw.maxCount {
		sw.mu.Unlock()
		return fmt.Errorf("rate: requested tokens (%d) exceeds limit (%d)", n, sw.maxCount)
	}

	// Calculate wait time if needed
	if len(sw.timestamps)+n > sw.maxCount {
		// Need to wait for oldest requests to expire
		needToExpire := len(sw.timestamps) + n - sw.maxCount
		if needToExpire > len(sw.timestamps) {
			needToExpire = len(sw.timestamps)
		}
		oldestToKeep := sw.timestamps[needToExpire-1]
		waitUntil := oldestToKeep.Add(sw.window).Add(time.Millisecond)
		sw.mu.Unlock()

		select {
		case <-time.After(time.Until(waitUntil)):
			return sw.WaitN(ctx, n) // Retry
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// We have capacity
	for i := 0; i < n; i++ {
		sw.timestamps = append(sw.timestamps, now)
	}
	sw.mu.Unlock()
	return nil
}

func (sw *SlidingWindowLimiter) Limit() Limit {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	return sw.limit
}

func (sw *SlidingWindowLimiter) SetLimit(newLimit Limit) {
	sw.SetLimitAt(time.Now(), newLimit)
}

func (sw *SlidingWindowLimiter) SetLimitAt(t time.Time, newLimit Limit) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	sw.cleanup(t)
	sw.limit = newLimit
	if newLimit > 0 {
		sw.window = time.Duration(float64(time.Second) * float64(sw.maxCount) / float64(newLimit))
	}
}

func (sw *SlidingWindowLimiter) Burst() int {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	return sw.maxCount
}

func (sw *SlidingWindowLimiter) SetBurst(newBurst int) {
	sw.SetBurstAt(time.Now(), newBurst)
}

func (sw *SlidingWindowLimiter) SetBurstAt(t time.Time, newBurst int) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	sw.cleanup(t)
	sw.maxCount = newBurst
	if sw.limit > 0 {
		sw.window = time.Duration(float64(time.Second) * float64(newBurst) / float64(sw.limit))
	}
}

// Tokens returns remaining capacity in current window
func (sw *SlidingWindowLimiter) Tokens() float64 {
	return sw.TokensAt(time.Now())
}

func (sw *SlidingWindowLimiter) TokensAt(t time.Time) float64 {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	sw.cleanup(t)
	return float64(sw.maxCount - len(sw.timestamps))
}
