package usecase

import (
	"context"
	"fmt"
	"html/template"

	"github.com/TubagusAldiMY/kasku/notification-service/internal/domain/event"
	"github.com/TubagusAldiMY/kasku/notification-service/internal/infrastructure/email"
	"github.com/rs/zerolog"
)

// NotificationUseCase menangani pengiriman semua jenis notifikasi email.
// Mengimplementasikan messaging.NotificationHandler interface.
type NotificationUseCase struct {
	sender    email.Sender
	templates *template.Template
	baseURL   string
	log       zerolog.Logger
}

func NewNotificationUseCase(
	sender email.Sender,
	templates *template.Template,
	baseURL string,
	log zerolog.Logger,
) *NotificationUseCase {
	return &NotificationUseCase{
		sender:    sender,
		templates: templates,
		baseURL:   baseURL,
		log:       log,
	}
}

func (uc *NotificationUseCase) HandleUserRegistered(ctx context.Context, e event.UserRegisteredEvent) error {
	body, err := email.RenderTemplate(uc.templates, "welcome.html", map[string]interface{}{
		"Username":         e.Username,
		"VerificationLink": fmt.Sprintf("%s/verify-email?token=%s", uc.baseURL, e.VerificationToken),
	})
	if err != nil {
		return fmt.Errorf("gagal render template welcome: %w", err)
	}

	if err := uc.sender.Send(e.Email, "Selamat Datang di KasKu! Verifikasi Email Anda", body); err != nil {
		return fmt.Errorf("gagal kirim welcome email ke %s: %w", maskEmail(e.Email), err)
	}
	uc.log.Info().Str("user_id", e.UserID).Msg("welcome email terkirim")
	return nil
}

func (uc *NotificationUseCase) HandleEmailVerificationResent(ctx context.Context, e event.EmailVerificationResentEvent) error {
	body, err := email.RenderTemplate(uc.templates, "verify_email.html", map[string]interface{}{
		"VerificationLink": fmt.Sprintf("%s/verify-email?token=%s", uc.baseURL, e.VerificationToken),
	})
	if err != nil {
		return fmt.Errorf("gagal render template verify_email: %w", err)
	}

	if err := uc.sender.Send(e.Email, "Verifikasi Email KasKu Anda", body); err != nil {
		return fmt.Errorf("gagal kirim verification email ke %s: %w", maskEmail(e.Email), err)
	}
	uc.log.Info().Str("user_id", e.UserID).Msg("verification email terkirim")
	return nil
}

func (uc *NotificationUseCase) HandlePasswordResetRequested(ctx context.Context, e event.PasswordResetRequestedEvent) error {
	body, err := email.RenderTemplate(uc.templates, "reset_password.html", map[string]interface{}{
		"ResetLink": fmt.Sprintf("%s/reset-password?token=%s", uc.baseURL, e.ResetToken),
	})
	if err != nil {
		return fmt.Errorf("gagal render template reset_password: %w", err)
	}

	if err := uc.sender.Send(e.Email, "Reset Password KasKu Anda", body); err != nil {
		return fmt.Errorf("gagal kirim reset password email ke %s: %w", maskEmail(e.Email), err)
	}
	uc.log.Info().Str("user_id", e.UserID).Msg("reset password email terkirim")
	return nil
}

func (uc *NotificationUseCase) HandlePaymentSucceeded(ctx context.Context, e event.PaymentSucceededEvent) error {
	body, err := email.RenderTemplate(uc.templates, "payment_success.html", map[string]interface{}{
		"PlanName":  e.PlanName,
		"AmountIDR": e.AmountIDR,
		"OrderID":   e.OrderID,
	})
	if err != nil {
		return fmt.Errorf("gagal render template payment_success: %w", err)
	}

	if err := uc.sender.Send(e.Email, "Pembayaran KasKu Berhasil", body); err != nil {
		return fmt.Errorf("gagal kirim payment success email ke %s: %w", maskEmail(e.Email), err)
	}
	uc.log.Info().Str("user_id", e.UserID).Str("order_id", e.OrderID).Msg("payment success email terkirim")
	return nil
}

func (uc *NotificationUseCase) HandlePaymentFailed(ctx context.Context, e event.PaymentFailedEvent) error {
	body, err := email.RenderTemplate(uc.templates, "payment_failed.html", map[string]interface{}{
		"OrderID": e.OrderID,
		"Reason":  e.Reason,
	})
	if err != nil {
		return fmt.Errorf("gagal render template payment_failed: %w", err)
	}

	if err := uc.sender.Send(e.Email, "Pembayaran KasKu Gagal", body); err != nil {
		return fmt.Errorf("gagal kirim payment failed email ke %s: %w", maskEmail(e.Email), err)
	}
	uc.log.Info().Str("user_id", e.UserID).Str("order_id", e.OrderID).Msg("payment failed email terkirim")
	return nil
}

func (uc *NotificationUseCase) HandleSubscriptionExpiring(ctx context.Context, e event.SubscriptionExpiringEvent) error {
	body, err := email.RenderTemplate(uc.templates, "subscription_expiring.html", map[string]interface{}{
		"PlanName":  e.PlanName,
		"ExpiresAt": e.ExpiresAt,
		"RenewLink": fmt.Sprintf("%s/billing", uc.baseURL),
	})
	if err != nil {
		return fmt.Errorf("gagal render template subscription_expiring: %w", err)
	}

	if err := uc.sender.Send(e.Email, "Subscription KasKu Anda Akan Segera Berakhir", body); err != nil {
		return fmt.Errorf("gagal kirim subscription expiring email ke %s: %w", maskEmail(e.Email), err)
	}
	uc.log.Info().Str("user_id", e.UserID).Str("plan", e.PlanName).Msg("subscription expiring email terkirim")
	return nil
}

func (uc *NotificationUseCase) HandleSubscriptionExpired(ctx context.Context, e event.SubscriptionExpiredEvent) error {
	body, err := email.RenderTemplate(uc.templates, "subscription_expired.html", map[string]interface{}{
		"PlanName":  e.PlanName,
		"RenewLink": fmt.Sprintf("%s/billing", uc.baseURL),
	})
	if err != nil {
		return fmt.Errorf("gagal render template subscription_expired: %w", err)
	}

	if err := uc.sender.Send(e.Email, "Subscription KasKu Anda Telah Berakhir", body); err != nil {
		return fmt.Errorf("gagal kirim subscription expired email ke %s: %w", maskEmail(e.Email), err)
	}
	uc.log.Info().Str("user_id", e.UserID).Str("plan", e.PlanName).Msg("subscription expired email terkirim")
	return nil
}

func (uc *NotificationUseCase) HandleSubscriptionCancelled(ctx context.Context, e event.SubscriptionCancelledEvent) error {
	body, err := email.RenderTemplate(uc.templates, "subscription_cancelled.html", map[string]interface{}{
		"PlanName":    e.PlanName,
		"CancelledAt": e.CancelledAt,
		"RenewLink":   fmt.Sprintf("%s/billing", uc.baseURL),
	})
	if err != nil {
		return fmt.Errorf("gagal render template subscription_cancelled: %w", err)
	}

	if err := uc.sender.Send(e.Email, "Langganan KasKu Kamu Telah Dibatalkan", body); err != nil {
		return fmt.Errorf("gagal kirim subscription cancelled email ke %s: %w", maskEmail(e.Email), err)
	}
	uc.log.Info().Str("user_id", e.UserID).Str("plan", e.PlanName).Msg("subscription cancelled email terkirim")
	return nil
}

// maskEmail menyembunyikan bagian lokal email untuk logging aman (u***@domain.com).
// Tidak pernah meng-log email address secara penuh sesuai standar OWASP A02.
func maskEmail(emailAddr string) string {
	for i, c := range emailAddr {
		if c == '@' && i > 0 {
			return string(emailAddr[0]) + "***" + emailAddr[i:]
		}
	}
	return "***"
}
