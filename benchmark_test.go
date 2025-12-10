package rateflow

import (
	"context"
	"testing"
)

func BenchmarkTokenBucketAllow(b *testing.B) {
	lim := NewLimiter(TokenBucket, Limit(1000), 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lim.Allow()
	}
}

func BenchmarkLeakyBucketAllow(b *testing.B) {
	lim := NewLimiter(LeakyBucket, Limit(1000), 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lim.Allow()
	}
}

func BenchmarkSlidingWindowAllow(b *testing.B) {
	lim := NewLimiter(SlidingWindow, Limit(1000), 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lim.Allow()
	}
}

func BenchmarkFixedWindowAllow(b *testing.B) {
	lim := NewLimiter(FixedWindow, Limit(1000), 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lim.Allow()
	}
}

func BenchmarkTokenBucketAllowParallel(b *testing.B) {
	lim := NewLimiter(TokenBucket, Limit(1000), 100)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			lim.Allow()
		}
	})
}

func BenchmarkTokenBucketWait(b *testing.B) {
	lim := NewLimiter(TokenBucket, Limit(10000), 1)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lim.Wait(ctx)
	}
}

func BenchmarkTokenBucketReserve(b *testing.B) {
	lim := NewLimiter(TokenBucket, Limit(1000), 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := lim.Reserve()
		_ = r.OK()
	}
}

func BenchmarkAllAlgorithmsComparison(b *testing.B) {
	algorithms := []struct {
		name string
		lim  Limiter
	}{
		{"TokenBucket", NewLimiter(TokenBucket, Limit(1000), 100)},
		{"LeakyBucket", NewLimiter(LeakyBucket, Limit(1000), 100)},
		{"SlidingWindow", NewLimiter(SlidingWindow, Limit(1000), 100)},
		{"FixedWindow", NewLimiter(FixedWindow, Limit(1000), 100)},
	}

	for _, algo := range algorithms {
		b.Run(algo.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				algo.lim.Allow()
			}
		})
	}
}
