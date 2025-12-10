package limiter

import "time"

// Reservation holds information about a reserved rate limit event
type Reservation struct {
	ok        bool
	lim       Limiter
	tokens    int
	timeToAct time.Time
	limit     Limit
}

// OK returns whether the reservation is valid
func (r *Reservation) OK() bool {
	return r.ok
}

// Delay returns how long to wait before the reserved event
func (r *Reservation) Delay() time.Duration {
	return r.DelayFrom(time.Now())
}

// DelayFrom returns the delay from the given time
func (r *Reservation) DelayFrom(t time.Time) time.Duration {
	if !r.ok {
		return -1
	}
	delay := r.timeToAct.Sub(t)
	if delay < 0 {
		return 0
	}
	return delay
}

// Cancel cancels the reservation (best effort)
func (r *Reservation) Cancel() {
	r.CancelAt(time.Now())
}

// CancelAt cancels the reservation at the given time (best effort)
func (r *Reservation) CancelAt(t time.Time) {
	if !r.ok {
		return
	}
	// Note: Not all algorithms can properly restore tokens
	// This is a best-effort operation
}
