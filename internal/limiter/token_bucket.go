package limiter

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"
)

// TokenBucketLimiter implements the token bucket algorithm
// Fully compatible with stdlib rate limiter
type TokenBucketLimiter struct {
	mu          sync.Mutex
	limit       Limit
	burst       int
	tokens      float64
	lastUpdated time.Time
}

// NewTokenBucket creates a new token bucket limiter
func NewTokenBucket(r Limit, b int) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		limit:       r,
		burst:       b,
		tokens:      float64(b),
		lastUpdated: time.Now(),
	}
}

func (tb *TokenBucketLimiter) Algorithm() Algorithm {
	return TokenBucket
}

func (tb *TokenBucketLimiter) Capabilities() Capabilities {
	return Capabilities{
		SupportsTokens:      true,
		SupportsBurst:       true,
		SupportsReservation: true,
	}
}

// advance updates the token count based on elapsed time
func (tb *TokenBucketLimiter) advance(now time.Time) {
	elapsed := now.Sub(tb.lastUpdated)
	tb.lastUpdated = now

	if tb.limit == Limit(math.MaxFloat64) {
		tb.tokens = float64(tb.burst)
		return
	}

	// Add tokens based on elapsed time
	delta := float64(tb.limit) * elapsed.Seconds()
	tb.tokens = math.Min(tb.tokens+delta, float64(tb.burst))
}

func (tb *TokenBucketLimiter) Allow() bool {
	return tb.AllowN(time.Now(), 1)
}

func (tb *TokenBucketLimiter) AllowN(t time.Time, n int) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.advance(t)

	if tb.tokens >= float64(n) {
		tb.tokens -= float64(n)
		return true
	}
	return false
}

func (tb *TokenBucketLimiter) Reserve() *Reservation {
	return tb.ReserveN(time.Now(), 1)
}

func (tb *TokenBucketLimiter) ReserveN(t time.Time, n int) *Reservation {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.advance(t)

	if n > tb.burst {
		return &Reservation{ok: false}
	}

	// Calculate wait time
	tokens := tb.tokens
	waitDuration := time.Duration(0)

	if tokens < float64(n) {
		needed := float64(n) - tokens
		if tb.limit > 0 {
			waitDuration = time.Duration(needed/float64(tb.limit)*float64(time.Second)) + time.Nanosecond
		}
	}

	tb.tokens -= float64(n)

	return &Reservation{
		ok:        true,
		lim:       tb,
		tokens:    n,
		timeToAct: t.Add(waitDuration),
		limit:     tb.limit,
	}
}

func (tb *TokenBucketLimiter) Wait(ctx context.Context) error {
	return tb.WaitN(ctx, 1)
}

func (tb *TokenBucketLimiter) WaitN(ctx context.Context, n int) error {
	r := tb.ReserveN(time.Now(), n)
	if !r.OK() {
		return fmt.Errorf("rate: requested tokens (%d) exceeds burst (%d)", n, tb.Burst())
	}

	delay := r.Delay()
	if delay == 0 {
		return nil
	}

	select {
	case <-time.After(delay):
		return nil
	case <-ctx.Done():
		r.Cancel()
		return ctx.Err()
	}
}

func (tb *TokenBucketLimiter) Limit() Limit {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	return tb.limit
}

func (tb *TokenBucketLimiter) SetLimit(newLimit Limit) {
	tb.SetLimitAt(time.Now(), newLimit)
}

func (tb *TokenBucketLimiter) SetLimitAt(t time.Time, newLimit Limit) {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.advance(t)
	tb.limit = newLimit
}

func (tb *TokenBucketLimiter) Burst() int {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	return tb.burst
}

func (tb *TokenBucketLimiter) SetBurst(newBurst int) {
	tb.SetBurstAt(time.Now(), newBurst)
}

func (tb *TokenBucketLimiter) SetBurstAt(t time.Time, newBurst int) {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.advance(t)
	tb.burst = newBurst
	if tb.tokens > float64(newBurst) {
		tb.tokens = float64(newBurst)
	}
}

func (tb *TokenBucketLimiter) Tokens() float64 {
	return tb.TokensAt(time.Now())
}

func (tb *TokenBucketLimiter) TokensAt(t time.Time) float64 {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.advance(t)
	return tb.tokens
}
