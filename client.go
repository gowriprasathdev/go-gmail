package gmail

import (
	"context"
	"fmt"
	"sync"

	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// Sender defines the interface for sending email messages.
// This allows for easier mocking in tests.
type Sender interface {
	NewMessage() *Message
}

// Client is a safe, concurrent Gmail API client configured with strict rate limiting
// to protect the user's account from quota bans.
type Client struct {
	srv       *gmail.Service
	limiter   *RateLimiter
	autoRetry bool
	mu        sync.RWMutex // Protects state if needed for concurrent modifications
}

// ClientOption defines an option for customizing the Client.
type ClientOption func(*Client)

// WithRateLimiter allows overriding the default rate limiter.
func WithRateLimiter(rl *RateLimiter) ClientOption {
	return func(c *Client) {
		c.limiter = rl
	}
}

// WithAutoRetry enables automatic waiting when rate limits (internal or 429) are hit.
func WithAutoRetry() ClientOption {
	return func(c *Client) {
		c.autoRetry = true
	}
}

// NewClient initializes a new Gmail client using standard Google OAuth2 JSON credentials
// (either User OAuth client credentials or a Service Account JSON).
func NewClient(ctx context.Context, credentialsJSON []byte, opts ...ClientOption) (*Client, error) {
	// The option.WithCredentialsJSON handles both Service Accounts and OAuth2 client flows.
	srv, err := gmail.NewService(ctx, option.WithCredentialsJSON(credentialsJSON), option.WithScopes(gmail.GmailSendScope))
	if err != nil {
		return nil, fmt.Errorf("failed to create gmail service: %w", err)
	}

	c := &Client{
		srv:     srv,
		limiter: DefaultRateLimiter(),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

// NewMessage starts the fluent builder pattern for composing a new email.
func (c *Client) NewMessage() *Message {
	return &Message{
		client: c,
	}
}
