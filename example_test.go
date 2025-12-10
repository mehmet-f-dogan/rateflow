package rateflow

import (
	"context"
	"fmt"
	"time"

	"github.com/mehmet-f-dogan/rateflow/internal/limiter"
)

func ExampleTokenBucket() {
	// Create a limiter: 10 requests/second, burst of 5
	limiter := NewLimiter(TokenBucket, limiter.Limit(10), 5)

	// Check if request is allowed
	if limiter.Allow() {
		fmt.Println("Request allowed")
	} else {
		fmt.Println("Request denied")
	}

	// Output: Request allowed
}

func ExampleLimiter_Wait() {
	limiter := NewLimiter(TokenBucket, limiter.Limit(2), 1)

	// Use all available tokens
	limiter.Allow()

	// Wait for next available token (with timeout)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := limiter.Wait(ctx); err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("Proceeding with request")
	}

	// Output: Proceeding with request
}

func ExampleLimiter_Reserve() {
	limiter := NewLimiter(TokenBucket, limiter.Limit(10), 5)

	// Reserve 3 tokens
	r := limiter.ReserveN(time.Now(), 3)
	if !r.OK() {
		fmt.Println("Reservation failed")
		return
	}

	// Check how long to wait
	delay := r.Delay()
	if delay > 0 {
		fmt.Printf("Wait for %v before proceeding", delay)
		time.Sleep(delay)
	}

	fmt.Println("Ready to proceed")
	// Output: Ready to proceed
}

func ExampleSlidingWindow() {
	// Sliding window: max 100 requests per 10 seconds
	limiter := NewLimiter(SlidingWindow, limiter.Limit(10), 100)

	allowed := 0
	for i := 0; i < 150; i++ {
		if limiter.Allow() {
			allowed++
		}
	}

	fmt.Printf("Allowed %d out of 150 requests", allowed)
	// Output: Allowed 100 out of 150 requests
}

func ExampleLimiter_SetLimit() {
	lm := NewLimiter(TokenBucket, limiter.Limit(10), 5)

	fmt.Printf("Initial limit: %v\n", lm.Limit())

	// Dynamically adjust rate limit
	lm.SetLimit(limiter.Limit(20))

	fmt.Printf("New limit: %v", lm.Limit())
	// Output:
	// Initial limit: 10
	// New limit: 20
}

func ExamplePerMinute() {
	// Create a limiter for 60 requests per minute
	limiter := NewLimiter(
		TokenBucket,
		PerMinute(60),
		10,
	)

	if limiter.Allow() {
		fmt.Println("API call allowed")
	}
	// Output: API call allowed
}

func ExampleCapabilities() {
	limiters := []limiter.Limiter{
		NewLimiter(TokenBucket, limiter.Limit(10), 5),
		NewLimiter(SlidingWindow, limiter.Limit(10), 5),
	}

	for _, lim := range limiters {
		caps := lim.Capabilities()
		fmt.Printf("%s - Tokens: %v, Burst: %v, Reservation: %v\n",
			lim.Algorithm(),
			caps.SupportsTokens,
			caps.SupportsBurst,
			caps.SupportsReservation,
		)
	}
	// Output:
	// TokenBucket - Tokens: true, Burst: true, Reservation: true
	// SlidingWindow - Tokens: false, Burst: false, Reservation: false
}
