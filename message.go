package gmail

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/googleapi"
)

// Attachment represents a file to be attached to the email.
type Attachment struct {
	Filename string
	Data     []byte
}

// Message represents an email being built.
type Message struct {
	client *Client

	to          []string
	cc          []string
	bcc         []string
	subject     string
	textBody    string
	htmlBody    string
	attachments []Attachment
}

// To adds a primary recipient. Can be chained.
func (m *Message) To(addresses ...string) *Message {
	m.to = append(m.to, addresses...)
	return m
}

// CC adds a Carbon Copy recipient.
func (m *Message) CC(addresses ...string) *Message {
	m.cc = append(m.cc, addresses...)
	return m
}

// BCC adds a Blind Carbon Copy recipient.
func (m *Message) BCC(addresses ...string) *Message {
	m.bcc = append(m.bcc, addresses...)
	return m
}

// Subject sets the email subject.
func (m *Message) Subject(sub string) *Message {
	m.subject = sub
	return m
}

// TextBody sets the plain text body of the email.
func (m *Message) TextBody(body string) *Message {
	m.textBody = body
	return m
}

// HTMLBody sets the HTML body of the email.
func (m *Message) HTMLBody(body string) *Message {
	m.htmlBody = body
	return m
}

// Attach adds an attachment from an io.Reader.
func (m *Message) Attach(filename string, r io.Reader) *Message {
	data, err := io.ReadAll(r)
	if err == nil {
		m.attachments = append(m.attachments, Attachment{
			Filename: filename,
			Data:     data,
		})
	}
	return m
}

// AttachFile adds an attachment by reading from the local file system.
func (m *Message) AttachFile(path string) *Message {
	data, err := os.ReadFile(path)
	if err == nil {
		m.attachments = append(m.attachments, Attachment{
			Filename: filepath.Base(path),
			Data:     data,
		})
	}
	return m
}

// Send finalizes the message, constructs the MIME payload, applies rate-limiting,
// and dispatches the email via the Gmail API. Return safely on rate limits.
func (m *Message) Send(ctx context.Context) error {
	if len(m.to) == 0 && len(m.cc) == 0 && len(m.bcc) == 0 {
		return ErrMissingRecipient
	}
	if m.textBody == "" && m.htmlBody == "" {
		return ErrMissingBody
	}
	if m.client == nil {
		return ErrClientNil
	}

	// 1. Check Rate Limiter
	if err := m.client.limiter.RequestToken(ctx); err != nil {
		return err
	}

	// 2. Build MIME Payload
	rawMesg, err := m.buildMIME()
	if err != nil {
		return fmt.Errorf("failed to build MIME payload: %w", err)
	}

	gMessage := &gmail.Message{
		Raw: rawMesg,
	}

	// 3. Dispatch & Handle 429 Interception
	_, err = m.client.srv.Users.Messages.Send("me", gMessage).Context(ctx).Do()
	if err != nil {
		return parseAPIError(err)
	}

	return nil
}

// buildMIME constructs the multipart MIME document and returns it base64URL-encoded (RFC 2822).
// The Gmail API expects exactly the standard Base64URLEncoding format.
func (m *Message) buildMIME() (string, error) {
	buf := &bytes.Buffer{}

	// Basic headers
	buf.WriteString("MIME-Version: 1.0\r\n")
	if len(m.to) > 0 {
		fmt.Fprintf(buf, "To: %s\r\n", strings.Join(m.to, ", "))
	}
	if len(m.cc) > 0 {
		fmt.Fprintf(buf, "Cc: %s\r\n", strings.Join(m.cc, ", "))
	}
	if len(m.bcc) > 0 {
		fmt.Fprintf(buf, "Bcc: %s\r\n", strings.Join(m.bcc, ", "))
	}
	fmt.Fprintf(buf, "Subject: %s\r\n", m.subject)

	hasAttachments := len(m.attachments) > 0

	var mixedWriter *multipart.Writer
	var bodyWriter interface{} = buf

	if hasAttachments {
		mixedWriter = multipart.NewWriter(buf)
		fmt.Fprintf(buf, "Content-Type: multipart/mixed; boundary=\"%s\"\r\n\r\n", mixedWriter.Boundary())
		bodyWriter = mixedWriter
	}

	// Internal part handling
	if m.textBody != "" && m.htmlBody != "" {
		// Both text and html, use multipart/alternative
		altBuf := &bytes.Buffer{}
		altWriter := multipart.NewWriter(altBuf)

		writeTextPart(altWriter, "text/plain", m.textBody)
		writeTextPart(altWriter, "text/html", m.htmlBody)
		altWriter.Close()

		if mw, ok := bodyWriter.(*multipart.Writer); ok {
			h := make(textproto.MIMEHeader)
			h.Set("Content-Type", fmt.Sprintf("multipart/alternative; boundary=\"%s\"", altWriter.Boundary()))
			part, _ := mw.CreatePart(h)
			part.Write(altBuf.Bytes())
		} else {
			fmt.Fprintf(buf, "Content-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n", altWriter.Boundary())
			buf.Write(altBuf.Bytes())
		}
	} else if m.htmlBody != "" {
		writeSinglePart(bodyWriter, "text/html", m.htmlBody)
	} else {
		writeSinglePart(bodyWriter, "text/plain", m.textBody)
	}

	if hasAttachments {
		for _, attachment := range m.attachments {
			h := make(textproto.MIMEHeader)
			h.Set("Content-Type", fmt.Sprintf("application/octet-stream; name=\"%s\"", attachment.Filename))
			h.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", attachment.Filename))
			h.Set("Content-Transfer-Encoding", "base64")

			part, err := mixedWriter.CreatePart(h)
			if err != nil {
				return "", err
			}

			// Encode attachment to base64 line-by-line (chunk size 76)
			b64Data := make([]byte, base64.StdEncoding.EncodedLen(len(attachment.Data)))
			base64.StdEncoding.Encode(b64Data, attachment.Data)

			chunkSize := 76
			for i := 0; i < len(b64Data); i += chunkSize {
				end := i + chunkSize
				if end > len(b64Data) {
					end = len(b64Data)
				}
				part.Write(b64Data[i:end])
				part.Write([]byte("\r\n"))
			}
		}
		mixedWriter.Close()
	}

	// Gmail requires Base64URL encoding
	return base64.URLEncoding.EncodeToString(buf.Bytes()), nil
}

func writeTextPart(w *multipart.Writer, contentType, body string) {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Type", fmt.Sprintf("%s; charset=\"UTF-8\"", contentType))
	part, _ := w.CreatePart(h)
	part.Write([]byte(body))
}

func writeSinglePart(w interface{}, contentType, body string) {
	if mw, ok := w.(*multipart.Writer); ok {
		h := make(textproto.MIMEHeader)
		h.Set("Content-Type", fmt.Sprintf("%s; charset=\"UTF-8\"", contentType))
		part, _ := mw.CreatePart(h)
		part.Write([]byte(body))
	} else {
		// w is bytes.Buffer
		buf := w.(*bytes.Buffer)
		fmt.Fprintf(buf, "Content-Type: %s; charset=\"UTF-8\"\r\n\r\n", contentType)
		buf.WriteString(body)
	}
}

// parseAPIError intercepts 429 Too Many Requests and Quota errs.
func parseAPIError(err error) error {
	var gErr *googleapi.Error
	if errors.As(err, &gErr) {
		if gErr.Code == http.StatusTooManyRequests { // 429
			// A 429 means we are hitting peak burst. Back off for a reasonable time.
			// Ideally, Google sends a Retry-After header, but googleapi.Error doesn't easily expose headers.
			// Let's assume a 10 second backoff for HTTP 429.
			return &ErrRateLimitExceeded{
				RetryAfter: time.Now().Add(10 * time.Second),
				Message:    "gmail API returned 429 Too Many Requests",
			}
		}
		if gErr.Code == http.StatusForbidden || gErr.Code == http.StatusBadRequest {
			// Sometimes quota errors are 403. Check message.
			msg := strings.ToLower(gErr.Message)
			if strings.Contains(msg, "quota") || strings.Contains(msg, "rate limit") {
				return ErrQuotaReached
			}
		}
	}
	return fmt.Errorf("gmail api error: %w", err)
}
