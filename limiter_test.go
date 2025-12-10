package rateflow

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestAllAlgorithms(t *testing.T) {
	algorithms := []struct {
		name string
		algo Algorithm
	}{
		{"TokenBucket", TokenBucket},
		{"LeakyBucket", LeakyBucket},
		{"SlidingWindow", SlidingWindow},
		{"FixedWindow", FixedWindow},
	}

	for _, test := range algorithms {
		t.Run(test.name, func(t *testing.T) {
			testBasicAllow(t, test.algo)
			testAllowN(t, test.algo)
			testWait(t, test.algo)
			testBurst(t, test.algo)
			testSetLimit(t, test.algo)
			testConcurrency(t, test.algo)
		})
	}
}

func testBasicAllow(t *testing.T, algo Algorithm) {
	lim := NewLimiter(algo, Limit(10), 5)

	// Should allow initial requests up to burst
	for i := 0; i < 5; i++ {
		if !lim.Allow() {
			t.Errorf("%s: expected Allow() = true for request %d", algo, i)
		}
	}

	// Should deny requests beyond burst
	if lim.Allow() {
		t.Errorf("%s: expected Allow() = false after burst exhausted", algo)
	}
}

func testAllowN(t *testing.T, algo Algorithm) {
	lim := NewLimiter(algo, Limit(10), 10)

	if !lim.AllowN(time.Now(), 5) {
		t.Errorf("%s: expected AllowN(5) = true", algo)
	}

	if !lim.AllowN(time.Now(), 5) {
		t.Errorf("%s: expected AllowN(5) = true", algo)
	}

	if lim.AllowN(time.Now(), 1) {
		t.Errorf("%s: expected AllowN(1) = false after exhausting tokens", algo)
	}
}

func testWait(t *testing.T, algo Algorithm) {
	lim := NewLimiter(algo, Limit(100), 1)

	// Exhaust burst
	if !lim.Allow() {
		t.Fatalf("%s: expected first Allow() = true", algo)
	}

	// Wait should succeed after short delay
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := lim.Wait(ctx)
	if err != nil {
		t.Errorf("%s: unexpected error from Wait(): %v", algo, err)
	}
}

func testBurst(t *testing.T, algo Algorithm) {
	lim := NewLimiter(algo, Limit(1), 5)

	if burst := lim.Burst(); burst != 5 {
		t.Errorf("%s: expected Burst() = 5, got %d", algo, burst)
	}

	lim.SetBurst(10)

	if burst := lim.Burst(); burst != 10 {
		t.Errorf("%s: expected Burst() = 10 after SetBurst, got %d", algo, burst)
	}
}

func testSetLimit(t *testing.T, algo Algorithm) {
	lim := NewLimiter(algo, Limit(10), 5)

	if limit := lim.Limit(); limit != Limit(10) {
		t.Errorf("%s: expected Limit() = 10, got %v", algo, limit)
	}

	lim.SetLimit(Limit(20))

	if limit := lim.Limit(); limit != Limit(20) {
		t.Errorf("%s: expected Limit() = 20 after SetLimit, got %v", algo, limit)
	}
}

func testConcurrency(t *testing.T, algo Algorithm) {
	lim := NewLimiter(algo, Limit(100), 50)
	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	// Launch 100 concurrent requests
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if lim.Allow() {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// Should allow exactly burst amount
	if successCount > 50 {
		t.Errorf("%s: concurrent test allowed %d requests, expected <= 50", algo, successCount)
	}
}

func TestTokenBucketTokens(t *testing.T) {
	lim := NewLimiter(TokenBucket, Limit(10), 10)

	tokens := lim.Tokens()
	if tokens != 10 {
		t.Errorf("expected initial tokens = 10, got %f", tokens)
	}

	lim.AllowN(time.Now(), 5)
	tokens = lim.Tokens()
	if tokens <= 4.5 || tokens >= 5.5 {
		t.Errorf("expected tokens = 5 after consuming 5, got %f", tokens)
	}
}

func TestReservation(t *testing.T) {
	lim := NewLimiter(TokenBucket, Limit(10), 5)

	// Exhaust tokens
	lim.AllowN(time.Now(), 5)

	// Reserve should work even when no tokens available
	r := lim.Reserve()
	if !r.OK() {
		t.Fatal("expected reservation to be OK")
	}

	delay := r.Delay()
	if delay <= 0 {
		t.Errorf("expected positive delay, got %v", delay)
	}

	// Cancel should not panic
	r.Cancel()
}

func TestContextCancellation(t *testing.T) {
	lim := NewLimiter(TokenBucket, Limit(1), 1)
	lim.Allow() // Exhaust

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := lim.Wait(ctx)
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestInfiniteLimit(t *testing.T) {
	lim := NewLimiter(TokenBucket, Inf, 1)

	// Should always allow
	for i := 0; i < 1000; i++ {
		if !lim.Allow() {
			t.Errorf("infinite limiter denied request %d", i)
		}
	}
}

func TestCapabilities(t *testing.T) {
	tests := []struct {
		algo              Algorithm
		expectTokens      bool
		expectBurst       bool
		expectReservation bool
	}{
		{TokenBucket, true, true, true},
		{LeakyBucket, false, false, true},
		{SlidingWindow, false, false, false},
		{FixedWindow, false, false, false},
	}

	for _, test := range tests {
		lim := NewLimiter(test.algo, Limit(10), 5)
		caps := lim.Capabilities()

		if caps.SupportsTokens != test.expectTokens {
			t.Errorf("%s: expected SupportsTokens=%v, got %v",
				test.algo, test.expectTokens, caps.SupportsTokens)
		}
		if caps.SupportsBurst != test.expectBurst {
			t.Errorf("%s: expected SupportsBurst=%v, got %v",
				test.algo, test.expectBurst, caps.SupportsBurst)
		}
		if caps.SupportsReservation != test.expectReservation {
			t.Errorf("%s: expected SupportsReservation=%v, got %v",
				test.algo, test.expectReservation, caps.SupportsReservation)
		}
	}
}
