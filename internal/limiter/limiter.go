package limiter

import (
	"context"
	"time"
)

type Limiter interface {
	// Core methods - all algorithms support these
	Allow() bool
	AllowN(t time.Time, n int) bool
	Wait(ctx context.Context) error
	WaitN(ctx context.Context, n int) error

	// Configuration methods
	Limit() Limit
	SetLimit(newLimit Limit)
	SetLimitAt(t time.Time, newLimit Limit)
	Burst() int
	SetBurst(newBurst int)
	SetBurstAt(t time.Time, newBurst int)

	// Token methods - behavior varies by algorithm
	Tokens() float64
	TokensAt(t time.Time) float64

	// Reservation methods - not all algorithms support this
	Reserve() *Reservation
	ReserveN(t time.Time, n int) *Reservation

	// Metadata
	Algorithm() Algorithm
	Capabilities() Capabilities
}
