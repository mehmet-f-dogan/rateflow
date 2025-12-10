package limiter

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"
)

// LeakyBucketLimiter implements the leaky bucket algorithm
// Requests are queued and processed at a constant rate
type LeakyBucketLimiter struct {
	mu           sync.Mutex
	limit        Limit
	capacity     int
	queue        []time.Time
	lastLeakTime time.Time
}

// NewLeakyBucket creates a new leaky bucket limiter
func NewLeakyBucket(r Limit, capacity int) *LeakyBucketLimiter {
	return &LeakyBucketLimiter{
		limit:        r,
		capacity:     capacity,
		queue:        make([]time.Time, 0, capacity),
		lastLeakTime: time.Now(),
	}
}

func (lb *LeakyBucketLimiter) Algorithm() Algorithm {
	return LeakyBucket
}

func (lb *LeakyBucketLimiter) Capabilities() Capabilities {
	return Capabilities{
		SupportsTokens:      false,
		SupportsBurst:       false,
		SupportsReservation: true,
	}
}

// leak removes expired items from the queue
func (lb *LeakyBucketLimiter) leak(now time.Time) {
	if lb.limit == Limit(math.MaxFloat64) || len(lb.queue) == 0 {
		return
	}

	elapsed := now.Sub(lb.lastLeakTime)
	lb.lastLeakTime = now

	leakCount := int(float64(lb.limit) * elapsed.Seconds())
	if leakCount > len(lb.queue) {
		leakCount = len(lb.queue)
	}

	lb.queue = lb.queue[leakCount:]
}

func (lb *LeakyBucketLimiter) Allow() bool {
	return lb.AllowN(time.Now(), 1)
}

func (lb *LeakyBucketLimiter) AllowN(t time.Time, n int) bool {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	lb.leak(t)

	if len(lb.queue)+n <= lb.capacity {
		for i := 0; i < n; i++ {
			lb.queue = append(lb.queue, t)
		}
		return true
	}
	return false
}

func (lb *LeakyBucketLimiter) Reserve() *Reservation {
	return lb.ReserveN(time.Now(), 1)
}

func (lb *LeakyBucketLimiter) ReserveN(t time.Time, n int) *Reservation {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	lb.leak(t)

	if n > lb.capacity {
		return &Reservation{ok: false}
	}

	waitDuration := time.Duration(0)
	if len(lb.queue)+n > lb.capacity {
		overflow := len(lb.queue) + n - lb.capacity
		if lb.limit > 0 {
			waitDuration = time.Duration(float64(overflow)/float64(lb.limit)*float64(time.Second)) + time.Nanosecond
		}
	}

	for i := 0; i < n; i++ {
		lb.queue = append(lb.queue, t)
	}

	return &Reservation{
		ok:        true,
		lim:       lb,
		tokens:    n,
		timeToAct: t.Add(waitDuration),
		limit:     lb.limit,
	}
}

func (lb *LeakyBucketLimiter) Wait(ctx context.Context) error {
	return lb.WaitN(ctx, 1)
}

func (lb *LeakyBucketLimiter) WaitN(ctx context.Context, n int) error {
	r := lb.ReserveN(time.Now(), n)
	if !r.OK() {
		return fmt.Errorf("rate: requested tokens (%d) exceeds capacity (%d)", n, lb.Burst())
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

func (lb *LeakyBucketLimiter) Limit() Limit {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	return lb.limit
}

func (lb *LeakyBucketLimiter) SetLimit(newLimit Limit) {
	lb.SetLimitAt(time.Now(), newLimit)
}

func (lb *LeakyBucketLimiter) SetLimitAt(t time.Time, newLimit Limit) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.leak(t)
	lb.limit = newLimit
}

func (lb *LeakyBucketLimiter) Burst() int {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	return lb.capacity
}

func (lb *LeakyBucketLimiter) SetBurst(newBurst int) {
	lb.SetBurstAt(time.Now(), newBurst)
}

func (lb *LeakyBucketLimiter) SetBurstAt(t time.Time, newBurst int) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.leak(t)
	lb.capacity = newBurst
	if len(lb.queue) > newBurst {
		lb.queue = lb.queue[:newBurst]
	}
}

// Tokens returns remaining capacity (not true tokens)
func (lb *LeakyBucketLimiter) Tokens() float64 {
	return lb.TokensAt(time.Now())
}

func (lb *LeakyBucketLimiter) TokensAt(t time.Time) float64 {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.leak(t)
	return float64(lb.capacity - len(lb.queue))
}
