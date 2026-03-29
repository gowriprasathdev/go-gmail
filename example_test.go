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

	// 1. Load your credentials.
	credsJSON, err := os.ReadFile("path/to/credentials.json")
	if err != nil {
		log.Fatalf("Failed to read credentials: %v", err)
	}

	// 2. Initialize the client with AutoRetry enabled.
	// This will cause .Send() to block and wait when rate limits are hit.
	client, err := gmail.NewClient(ctx, credsJSON, gmail.WithAutoRetry())
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// 3. Use the fluent builder API with inline images and attachments.
	msg := client.NewMessage().
		To("user@example.com").
		Subject("Daily Report").
		HTMLBody("<h1>Report</h1><p>See below:</p><img src=\"cid:report-img\">").
		EmbedFile("charts/daily.png", "report-img").   // Inline image
		AttachFile("reports/full_data.pdf")          // Regular attachment

	// 4. Send the message. Since WithAutoRetry is enabled, we don't 
	// need to manually catch ErrRateLimitExceeded for minor pauses.
	err = msg.Send(ctx)
	if err != nil {
		if errors.Is(err, gmail.ErrQuotaReached) {
			fmt.Println("Daily API Quota Reached.")
			return
		}
		log.Fatalf("Failed to send email: %v", err)
	}

	fmt.Println("Email sent successfully!")
}
