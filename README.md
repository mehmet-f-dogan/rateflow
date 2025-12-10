# RateLimit - Production-Ready Rate Limiting for Go

A high-performance, thread-safe rate limiting library with multiple algorithms and stdlib compatibility.

## Features

- 🚀 **4 Algorithm Implementations**: Token Bucket, Leaky Bucket, Sliding Window, Fixed Window
- 🔒 **Thread-Safe**: All operations are safe for concurrent use
- 🔄 **Stdlib Compatible**: Drop-in replacement for `golang.org/x/time/rate`
- 📦 **Zero Dependencies**: Only uses Go standard library
- ✅ **100% Test Coverage**: Comprehensive tests including race conditions
- ⚡ **Benchmarked**: Performance tested and optimized
- 📚 **Well Documented**: Clear examples and API documentation

## Installation

```bash
go get github.com/mehmet-f-dogan/rateflow
```

## Quick Start

```go
import "github.com/mehmet-f-dogan/rateflow"

// Create a token bucket limiter: 10 requests/second, burst of 5
limiter := rateflow.NewLimiter(rateflow.TokenBucket, rateflow.Limit(10), 5)

// Check if request is allowed
if limiter.Allow() {
    // Process request
}

// Wait for permission (blocking)
if err := limiter.Wait(context.Background()); err != nil {
    // Handle error
}
```

## Algorithm Comparison

| Algorithm      | Best For                        | Tokens() | Burst() | Reserve() |
| -------------- | ------------------------------- | -------- | ------- | --------- |
| Token Bucket   | General purpose, bursty traffic | ✅       | ✅      | ✅        |
| Leaky Bucket   | Smooth rate limiting            | ⚠️       | ⚠️      | ⚠️        |
| Sliding Window | Precise window-based limits     | ⚠️       | ⚠️      | ❌        |
| Fixed Window   | Simple time-based limits        | ⚠️       | ⚠️      | ❌        |

✅ Fully supported | ⚠️ Limited support | ❌ Not supported

## License

MIT
