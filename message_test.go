package gmail

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestBuildMIME_ContentType(t *testing.T) {
	m := &Message{
		to:      []string{"to@example.com"},
		subject: "Test",
		htmlBody: "<h1>Hello</h1>",
		attachments: []Attachment{
			{
				Filename: "test.png",
				Data:     []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01"), // Fake PNG header
			},
		},
	}

	raw, err := m.buildMIME()
	if err != nil {
		t.Fatalf("buildMIME failed: %v", err)
	}

	decoded, _ := base64.URLEncoding.DecodeString(raw)
	body := string(decoded)

	if !strings.Contains(body, "Content-Type: image/png") {
		t.Errorf("Expected image/png Content-Type, not found in body:\n%s", body)
	}
}

func TestMaxMessageSize(t *testing.T) {
	m := &Message{
		textBody: strings.Repeat("a", MaxMessageSize + 1),
		to:       []string{"to@example.com"},
		client: &Client{
			limiter: DefaultRateLimiter(),
		},
	}

	err := m.sendOnce(nil)
	if err == nil || !strings.Contains(err.Error(), "exceeds Gmail limit") {
		t.Errorf("Expected size limit error, got: %v", err)
	}
}
