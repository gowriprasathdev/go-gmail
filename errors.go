// Package gmail provides a fluent builder API for sending emails via the Gmail v1 API.
package gmail

import (
	"errors"
	"fmt"
	"time"
)

// Common validation errors.
var (
	ErrClientNil        = errors.New("gmail client is nil")
	ErrMissingRecipient = errors.New("message must have at least one recipient (To, Cc, or Bcc)")
	ErrMissingBody      = errors.New("message must have a text or HTML body")
	ErrQuotaReached     = errors.New("gmail API daily quota reached")
)

// ErrRateLimitExceeded is returned when a sending request is rejected due
// to exceeding either internal rate limits or Gmail API limits (HTTP 429).
type ErrRateLimitExceeded struct {
	// RetryAfter specifies the absolute time when the caller may retry sending.
	RetryAfter time.Time

	// Message holds the underlying reason for the rate limit.
	Message string
}

// Error implements the error interface.
func (e *ErrRateLimitExceeded) Error() string {
	return fmt.Sprintf("rate limit exceeded: %s. Retry after %v", e.Message, e.RetryAfter.Format(time.RFC3339))
}
