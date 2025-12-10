package rateflow

import (
	"math"
	"time"

	"github.com/mehmet-f-dogan/rateflow/internal/limiter"
)

// Limit represents the rate of events per second
type Limit = limiter.Limit

// Inf is the infinite rate limit (no limit)
const Inf = Limit(math.MaxFloat64)

// Every converts a minimum time interval between events to a Limit
func Every(interval time.Duration) Limit {
	if interval <= 0 {
		return Inf
	}
	return 1 / Limit(interval.Seconds())
}

// PerSecond converts requests per second to a Limit
func PerSecond(n int) Limit {
	return Limit(n)
}

// PerMinute converts requests per minute to a Limit
func PerMinute(n int) Limit {
	return Limit(n) / 60
}

// PerHour converts requests per hour to a Limit
func PerHour(n int) Limit {
	return Limit(n) / 3600
}

// Algorithm represents the rate limiting algorithm type
type Algorithm = limiter.Algorithm

const (
	TokenBucket   Algorithm = limiter.TokenBucket
	LeakyBucket   Algorithm = limiter.LeakyBucket
	SlidingWindow Algorithm = limiter.SlidingWindow
	FixedWindow   Algorithm = limiter.FixedWindow
)

// Capabilities describes what features an algorithm supports
type Capabilities = limiter.Capabilities

// Limiter is the main interface compatible with golang.org/x/time/rate
type Limiter = limiter.Limiter

// Reservation holds information about a reserved rate limit event
type Reservation = limiter.Reservation

// NewLimiter creates a new rate limiter with the specified algorithm
func NewLimiter(algo Algorithm, r Limit, b int) Limiter {
	switch algo {
	case TokenBucket:
		return limiter.NewTokenBucket(r, b)
	case LeakyBucket:
		return limiter.NewLeakyBucket(r, b)
	case SlidingWindow:
		return limiter.NewSlidingWindow(r, b)
	case FixedWindow:
		return limiter.NewFixedWindow(r, b)
	default:
		return limiter.NewTokenBucket(r, b)
	}
}
