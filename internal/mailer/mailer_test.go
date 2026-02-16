package mailer

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/allyourbase/ayb/internal/testutil"
)

func TestLogMailerSend(t *testing.T) {
	logger := testutil.DiscardLogger()
	m := NewLogMailer(logger)

	err := m.Send(context.Background(), &Message{
		To:      "user@example.com",
		Subject: "Test Subject",
		HTML:    "<p>Hello</p>",
		Text:    "Hello",
	})
	testutil.NoError(t, err)
}

func TestWebhookMailerSend(t *testing.T) {
	var received webhookPayload
	var gotSig string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSig = r.Header.Get("X-AYB-Signature")
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	secret := "test-webhook-secret"
	m := NewWebhookMailer(WebhookConfig{
		URL:    srv.URL,
		Secret: secret,
	})

	msg := &Message{
		To:      "user@example.com",
		Subject: "Test",
		HTML:    "<p>Hi</p>",
		Text:    "Hi",
	}
	err := m.Send(context.Background(), msg)
	testutil.NoError(t, err)

	testutil.Equal(t, received.To, "user@example.com")
	testutil.Equal(t, received.Subject, "Test")
	testutil.Equal(t, received.HTML, "<p>Hi</p>")
	testutil.Equal(t, received.Text, "Hi")

	// Verify HMAC signature.
	testutil.True(t, gotSig != "", "signature header should be set")
	payload, _ := json.Marshal(received)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expectedSig := hex.EncodeToString(mac.Sum(nil))
	testutil.Equal(t, gotSig, expectedSig)
}

func TestWebhookMailerNoSecret(t *testing.T) {
	var gotSig string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSig = r.Header.Get("X-AYB-Signature")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	m := NewWebhookMailer(WebhookConfig{URL: srv.URL})
	err := m.Send(context.Background(), &Message{To: "a@b.com", Subject: "x"})
	testutil.NoError(t, err)
	testutil.Equal(t, gotSig, "")
}

func TestWebhookMailerDefaultTimeout(t *testing.T) {
	m := NewWebhookMailer(WebhookConfig{URL: "http://localhost"})
	testutil.Equal(t, m.client.Timeout.Seconds(), float64(10))
}

func TestWebhookMailerCustomTimeout(t *testing.T) {
	m := NewWebhookMailer(WebhookConfig{URL: "http://localhost", Timeout: 30 * time.Second})
	testutil.Equal(t, m.client.Timeout.Seconds(), float64(30))
}

func TestWebhookMailerNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	m := NewWebhookMailer(WebhookConfig{URL: srv.URL})
	err := m.Send(context.Background(), &Message{To: "a@b.com", Subject: "x"})
	testutil.ErrorContains(t, err, "status 500")
}

func TestRenderPasswordReset(t *testing.T) {
	html, text, err := RenderPasswordReset(TemplateData{
		AppName:   "MyApp",
		ActionURL: "https://example.com/reset?token=abc123",
	})
	testutil.NoError(t, err)
	testutil.Contains(t, html, "Reset your password")
	testutil.Contains(t, html, "MyApp")
	testutil.Contains(t, html, "https://example.com/reset?token=abc123")
	testutil.Contains(t, text, "Reset your password")
	testutil.True(t, len(text) > 0, "text fallback should not be empty")
}

func TestRenderVerification(t *testing.T) {
	html, text, err := RenderVerification(TemplateData{
		AppName:   "MyApp",
		ActionURL: "https://example.com/verify?token=xyz",
	})
	testutil.NoError(t, err)
	testutil.Contains(t, html, "Verify your email")
	testutil.Contains(t, html, "MyApp")
	testutil.Contains(t, html, "https://example.com/verify?token=xyz")
	testutil.Contains(t, text, "Verify your email")
}

func TestStripHTML(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain text", "hello world", "hello world"},
		{"simple tag", "<p>hello</p>", "hello"},
		{"nested tags", "<div><p>hello</p></div>", "hello"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripHTML(tt.in)
			testutil.Equal(t, got, tt.want)
		})
	}
}

func TestSMTPMailerConfigValidation(t *testing.T) {
	// Verify NewSMTPMailer creates without panic.
	m := NewSMTPMailer(SMTPConfig{
		Host: "smtp.example.com",
		Port: 587,
		From: "noreply@example.com",
	})
	testutil.True(t, m != nil, "SMTPMailer should not be nil")
}

func TestSMTPMailerFormatFrom(t *testing.T) {
	tests := []struct {
		name     string
		from     string
		fromName string
		want     string
	}{
		{"address only", "noreply@example.com", "", "noreply@example.com"},
		{"with display name", "noreply@example.com", "MyApp", "MyApp <noreply@example.com>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &SMTPMailer{cfg: SMTPConfig{From: tt.from, FromName: tt.fromName}}
			testutil.Equal(t, m.formatFrom(), tt.want)
		})
	}
}

func TestSMTPMailerAuthTypes(t *testing.T) {
	tests := []struct {
		method   string
		wantSame bool // true if should return same as PLAIN (the default)
	}{
		{"PLAIN", true},
		{"LOGIN", false},
		{"CRAM-MD5", false},
		{"", true}, // unknown defaults to PLAIN
	}

	plainMailer := &SMTPMailer{cfg: SMTPConfig{AuthMethod: "PLAIN"}}
	plainResult := plainMailer.authType()

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			m := &SMTPMailer{cfg: SMTPConfig{AuthMethod: tt.method}}
			result := m.authType()
			if tt.wantSame {
				testutil.Equal(t, result, plainResult)
			} else {
				testutil.True(t, result != plainResult,
					"expected %q to produce different auth type than PLAIN", tt.method)
			}
		})
	}

	// Verify LOGIN and CRAM-MD5 are distinct from each other.
	loginMailer := &SMTPMailer{cfg: SMTPConfig{AuthMethod: "LOGIN"}}
	cramMailer := &SMTPMailer{cfg: SMTPConfig{AuthMethod: "CRAM-MD5"}}
	testutil.True(t, loginMailer.authType() != cramMailer.authType(),
		"LOGIN and CRAM-MD5 should produce different auth types")
}
