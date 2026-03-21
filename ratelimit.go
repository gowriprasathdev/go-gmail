package gmail

import (
	"context"
	"time"

	"golang.org/x/time/rate"
)

// DefaultSendRate is the default number of emails allowed per second per user.
// We set this conservatively to 2 emails per second to avoid quota bans.
const DefaultSendRate rate.Limit = 2.0

// DefaultSendBurst is the default maximum burst size.
const DefaultSendBurst int = 2

// RateLimiter wraps the standard library rate.Limiter to provide custom wait logic
// and error returns as required by the library's safety guarantees.
type RateLimiter struct {
	limiter *rate.Limiter
}

// NewRateLimiter creates a new custom RateLimiter with the specified rate and burst.
func NewRateLimiter(r rate.Limit, b int) *RateLimiter {
	return &RateLimiter{
		limiter: rate.NewLimiter(r, b),
	}
}

// DefaultRateLimiter returns a RateLimiter configured with sensible conservative defaults
// designed to prevent the associated Gmail account from being blocked.
func DefaultRateLimiter() *RateLimiter {
	return NewRateLimiter(DefaultSendRate, DefaultSendBurst)
}

// RequestToken evaluates whether a request can be executed immediately.
// If a delay is required, it returns ErrRateLimitExceeded containing the RetryAfter time.
func (rl *RateLimiter) RequestToken(ctx context.Context) error {
	r := rl.limiter.Reserve()
	if !r.OK() {
		return &ErrRateLimitExceeded{
			RetryAfter: time.Now().Add(time.Second),
			Message:    "rate limiter capacity exceeded or configured burst is zero",
		}
	}

	delay := r.Delay()
	if delay == 0 {
		return nil
	}

	// A delay is required. Cancel the reservation and return a custom error
	// instead of blocking silently.
	r.Cancel()
	return &ErrRateLimitExceeded{
		RetryAfter: time.Now().Add(delay),
		Message:    "internal send rate limit exceeded",
	}
}
