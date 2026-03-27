package gmail_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/gowriprasathdev/go-gmail"
)

func Example() {
	ctx := context.Background()

	// 1. Load your credentials (e.g., from a Service Account JSON).
	credsJSON, err := os.ReadFile("path/to/credentials.json")
	if err != nil {
		log.Fatalf("Failed to read credentials: %v", err)
	}

	// 2. Initialize the client. This client is safe for concurrent use and
	// includes a built-in strict rate limiter default of ~2 req/sec.
	client, err := gmail.NewClient(ctx, credsJSON)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// 3. Use the fluent builder API to construct a message.
	msg := client.NewMessage().
		To("user@example.com").
		CC("boss@example.com").
		Subject("Daily Report").
		TextBody("Here is the daily report.").
		HTMLBody("<h1>Daily Report</h1><p>Here is the daily report.</p>")

	// 4. Send the message and handle potential rate limits gracefully.
	err = msg.Send(ctx)
	if err != nil {
		// Example of properly checking for the custom ErrRateLimitExceeded error.
		var rlErr *gmail.ErrRateLimitExceeded
		if errors.As(err, &rlErr) {
			fmt.Printf("Rate limit exceeded! Please pause operations until %v\n", rlErr.RetryAfter)
			// e.g., time.Sleep(time.Until(rlErr.RetryAfter))
			return
		}

		if errors.Is(err, gmail.ErrQuotaReached) {
			fmt.Println("Daily API Quota Reached. Terminating batch jobs.")
			return
		}

		log.Fatalf("Failed to send email: %v", err)
	}

	fmt.Println("Email sent successfully!")
}
