package limiter

type Limit float64

type Algorithm int

const (
	TokenBucket Algorithm = iota
	LeakyBucket
	SlidingWindow
	FixedWindow
)

func (a Algorithm) String() string {
	switch a {
	case TokenBucket:
		return "TokenBucket"
	case LeakyBucket:
		return "LeakyBucket"
	case SlidingWindow:
		return "SlidingWindow"
	case FixedWindow:
		return "FixedWindow"
	default:
		return "Unknown"
	}
}

type Capabilities struct {
	SupportsTokens      bool
	SupportsBurst       bool
	SupportsReservation bool
}
