package usecase

import (
	"context"
	"errors"
	"html/template"
	"io"
	"strings"
	"testing"

	"github.com/TubagusAldiMY/kasku/notification-service/internal/domain/event"
	"github.com/rs/zerolog"
)

// fakeSender adalah in-memory implementasi email.Sender untuk verifikasi
// rendering + recipient tanpa SMTP server.
type fakeSender struct {
	failOnce  bool
	calls     int
	lastTo    string
	lastSubj  string
	lastBody  string
	allBodies []string
}

func (f *fakeSender) Send(to, subject, htmlBody string) error {
	f.calls++
	f.lastTo, f.lastSubj, f.lastBody = to, subject, htmlBody
	f.allBodies = append(f.allBodies, htmlBody)
	if f.failOnce {
		f.failOnce = false
		return errors.New("smtp unavailable")
	}
	return nil
}

// makeTestTemplates merangkai template inline berisi placeholder semua field
// yang dipakai handler. Penulisan terpisah dari template HTML production
// supaya test tidak tergantung filesystem.
func makeTestTemplates(t *testing.T) *template.Template {
	t.Helper()
	tmpl := template.New("")
	defs := map[string]string{
		"welcome.html":              `welcome user={{.Username}} link={{.VerificationLink}}`,
		"verify_email.html":         `verify link={{.VerificationLink}}`,
		"reset_password.html":       `reset link={{.ResetLink}}`,
		"payment_success.html":      `paid plan={{.PlanName}} amount={{.AmountIDR}} order={{.OrderID}}`,
		"payment_failed.html":       `failed order={{.OrderID}} reason={{.Reason}}`,
		"subscription_expiring.html": `expiring plan={{.PlanName}} expires={{.ExpiresAt}} renew={{.RenewLink}}`,
		"subscription_expired.html":  `expired plan={{.PlanName}} renew={{.RenewLink}}`,
	}
	for name, body := range defs {
		if _, err := tmpl.New(name).Parse(body); err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
	}
	return tmpl
}

func newTestUC(t *testing.T, sender *fakeSender) *NotificationUseCase {
	t.Helper()
	return NewNotificationUseCase(sender, makeTestTemplates(t), "https://kasku.example", zerolog.New(io.Discard))
}

func TestHandleUserRegistered_SendsWelcomeWithVerificationLink(t *testing.T) {
	t.Parallel()
	s := &fakeSender{}
	uc := newTestUC(t, s)
	err := uc.HandleUserRegistered(context.Background(), event.UserRegisteredEvent{
		UserID:            "u1",
		Email:             "alice@example.com",
		Username:          "alice",
		VerificationToken: "tok-abc",
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if s.calls != 1 {
		t.Fatalf("expected 1 send, got %d", s.calls)
	}
	if s.lastTo != "alice@example.com" {
		t.Fatalf("wrong recipient: %s", s.lastTo)
	}
	if !strings.Contains(s.lastBody, "tok-abc") {
		t.Fatalf("body missing verification token: %s", s.lastBody)
	}
	if !strings.Contains(s.lastBody, "user=alice") {
		t.Fatalf("body missing username: %s", s.lastBody)
	}
}

func TestHandlePasswordResetRequested_SendsResetLink(t *testing.T) {
	t.Parallel()
	s := &fakeSender{}
	uc := newTestUC(t, s)
	err := uc.HandlePasswordResetRequested(context.Background(), event.PasswordResetRequestedEvent{
		UserID:     "u1",
		Email:      "bob@example.com",
		ResetToken: "reset-xyz",
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !strings.Contains(s.lastBody, "reset-xyz") {
		t.Fatalf("body missing reset token")
	}
	if !strings.Contains(s.lastSubj, "Reset Password") {
		t.Fatalf("wrong subject: %s", s.lastSubj)
	}
}

func TestHandlePaymentSucceeded_IncludesOrderAndAmount(t *testing.T) {
	t.Parallel()
	s := &fakeSender{}
	uc := newTestUC(t, s)
	err := uc.HandlePaymentSucceeded(context.Background(), event.PaymentSucceededEvent{
		UserID:    "u1",
		Email:     "c@example.com",
		OrderID:   "ORDER-1",
		AmountIDR: 99000,
		PlanName:  "PRO",
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !strings.Contains(s.lastBody, "ORDER-1") || !strings.Contains(s.lastBody, "99000") || !strings.Contains(s.lastBody, "PRO") {
		t.Fatalf("payment success body missing fields: %s", s.lastBody)
	}
}

func TestHandlePaymentFailed_IncludesReason(t *testing.T) {
	t.Parallel()
	s := &fakeSender{}
	uc := newTestUC(t, s)
	err := uc.HandlePaymentFailed(context.Background(), event.PaymentFailedEvent{
		UserID:  "u1",
		Email:   "d@example.com",
		OrderID: "ORDER-2",
		Reason:  "card declined",
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !strings.Contains(s.lastBody, "card declined") {
		t.Fatalf("body missing reason: %s", s.lastBody)
	}
}

func TestHandleSubscriptionExpiring_IncludesRenewLink(t *testing.T) {
	t.Parallel()
	s := &fakeSender{}
	uc := newTestUC(t, s)
	err := uc.HandleSubscriptionExpiring(context.Background(), event.SubscriptionExpiringEvent{
		UserID:    "u1",
		Email:     "e@example.com",
		PlanName:  "PRO",
		ExpiresAt: "2026-06-01",
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !strings.Contains(s.lastBody, "https://kasku.example/billing") {
		t.Fatalf("body missing renew link: %s", s.lastBody)
	}
}

func TestHandleSubscriptionExpired_StillIncludesRenewLink(t *testing.T) {
	t.Parallel()
	s := &fakeSender{}
	uc := newTestUC(t, s)
	err := uc.HandleSubscriptionExpired(context.Background(), event.SubscriptionExpiredEvent{
		UserID:   "u1",
		Email:    "f@example.com",
		PlanName: "FREE",
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !strings.Contains(s.lastBody, "/billing") {
		t.Fatalf("body missing renew link: %s", s.lastBody)
	}
}

func TestHandleEmailVerificationResent_SendsVerifyTemplate(t *testing.T) {
	t.Parallel()
	s := &fakeSender{}
	uc := newTestUC(t, s)
	err := uc.HandleEmailVerificationResent(context.Background(), event.EmailVerificationResentEvent{
		UserID:            "u1",
		Email:             "g@example.com",
		VerificationToken: "vtok",
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !strings.HasPrefix(s.lastBody, "verify link=") {
		t.Fatalf("wrong template used: %s", s.lastBody)
	}
}

func TestHandler_PropagatesSenderError(t *testing.T) {
	t.Parallel()
	s := &fakeSender{failOnce: true}
	uc := newTestUC(t, s)
	err := uc.HandleUserRegistered(context.Background(), event.UserRegisteredEvent{
		Email:    "h@example.com",
		Username: "h",
	})
	if err == nil {
		t.Fatal("expected error when sender fails")
	}
	// Error message wajib mengandung versi MASKED email (OWASP A02), bukan plaintext
	if strings.Contains(err.Error(), "h@example.com") {
		t.Fatalf("error message membocorkan full email PII: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "***") {
		t.Fatalf("expected masked email in error, got: %s", err.Error())
	}
}

// ---------- maskEmail (PII safety, OWASP A02) ----------

func TestMaskEmail(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input string
		want  string
	}{
		{"alice@example.com", "a***@example.com"},
		{"x@y.z", "x***@y.z"},
		{"no-at-sign", "***"},
		{"@noprefix.com", "***"},
		{"", "***"},
	}
	for _, c := range cases {
		got := maskEmail(c.input)
		if got != c.want {
			t.Errorf("maskEmail(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}
