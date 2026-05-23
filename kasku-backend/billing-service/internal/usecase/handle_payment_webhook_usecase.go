package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/TubagusAldiMY/kasku/billing-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/billing-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/billing-service/internal/domain/repository"
	"github.com/rs/zerolog"
)

const (
	// subscriptionActivePeriodDays adalah durasi subscription aktif setelah pembayaran sukses.
	subscriptionActivePeriodDays = 30

	// webhookEventPaymentSuccess adalah event name dari orchestrator saat pembayaran berhasil.
	webhookEventPaymentSuccess = "payment.success"

	// webhookEventPaymentFailed adalah event name dari orchestrator saat pembayaran ditolak.
	webhookEventPaymentFailed = "payment.failed"

	// webhookEventPaymentExpired adalah event name dari orchestrator saat window pembayaran habis.
	webhookEventPaymentExpired = "payment.expired"

	// outboxEventTypeSubscriptionActivated adalah event type yang disimpan di outbox.
	outboxEventTypeSubscriptionActivated = "subscription.activated"

	// routingKeySubscriptionActivated adalah routing key RabbitMQ untuk event aktivasi.
	routingKeySubscriptionActivated = "subscription.activated"
)

// PaymentWebhookInput adalah data yang diparsing dari payload webhook Payment Orchestrator.
type PaymentWebhookInput struct {
	Event             string
	ExternalPaymentID string
	RefID             string
	Amount            int
	Status            string
	PaidAt            *time.Time
}

// subscriptionActivatedEventPayload adalah payload yang disimpan ke outbox setelah subscription aktif.
// Consumer (notification-service, user-service) membaca event ini untuk follow-up action.
type subscriptionActivatedEventPayload struct {
	SubscriptionID string `json:"subscription_id"`
	UserID         string `json:"user_id"`
	PlanID         string `json:"plan_id"`
	PeriodEnd      string `json:"period_end"`
	ActivatedAt    string `json:"activated_at"`
}

// HandlePaymentWebhookUseCase mendefinisikan kontrak untuk memproses notifikasi webhook
// dari Payment Orchestrator.
//
//go:generate mockgen -source=$GOFILE -destination=../../tests/mocks/mock_handle_payment_webhook_usecase.go -package=mocks
type HandlePaymentWebhookUseCase interface {
	Execute(ctx context.Context, input PaymentWebhookInput) error
}

type handlePaymentWebhookUseCase struct {
	paymentRepo repository.PaymentRepository
	subRepo     repository.SubscriptionRepository
	log         zerolog.Logger
}

// NewHandlePaymentWebhookUseCase membuat instance use case webhook handler.
func NewHandlePaymentWebhookUseCase(
	paymentRepo repository.PaymentRepository,
	subRepo repository.SubscriptionRepository,
	log zerolog.Logger,
) HandlePaymentWebhookUseCase {
	return &handlePaymentWebhookUseCase{
		paymentRepo: paymentRepo,
		subRepo:     subRepo,
		log:         log,
	}
}

// Execute memproses webhook dari Payment Orchestrator dengan urutan:
//  1. Lookup payment berdasarkan RefID (termasuk PlanID yang sudah disimpan saat create)
//  2. Idempotency guard — skip jika status sudah final
//  3. Update status payment berdasarkan event
//  4. Jika PAID: buat/reuse subscription → aktifkan → link payment → outbox event
//  5. Jika FAILED/EXPIRED: update status saja, tidak ada perubahan subscription
func (uc *handlePaymentWebhookUseCase) Execute(ctx context.Context, input PaymentWebhookInput) error {
	// Step 1: Lookup payment — PlanID sudah tersimpan di record ini
	existingPayment, err := uc.paymentRepo.GetByExternalRefID(ctx, input.RefID)
	if err != nil {
		if err == domainerrors.ErrPaymentNotFound {
			// RefID tidak dikenal — bukan milik service ini, abaikan dengan aman
			uc.log.Warn().
				Str("ref_id", input.RefID).
				Str("event", input.Event).
				Msg("webhook diterima untuk refId yang tidak dikenal, diabaikan")
			return nil
		}
		return fmt.Errorf("gagal lookup payment untuk webhook refId %s: %w", input.RefID, err)
	}

	// Step 2: Idempotency guard — jika sudah di status final, tidak perlu proses lagi
	if existingPayment.Status.IsFinalStatus() {
		uc.log.Info().
			Str("payment_id", existingPayment.ID.String()).
			Str("ref_id", input.RefID).
			Str("current_status", string(existingPayment.Status)).
			Str("incoming_event", input.Event).
			Msg("webhook diabaikan: payment sudah di status final")
		return nil
	}

	// Step 3: Petakan event orchestrator ke domain PaymentStatus
	newStatus, isKnownEvent := resolvePaymentStatusFromWebhookEvent(input.Event, input.Status)
	if !isKnownEvent {
		uc.log.Warn().
			Str("event", input.Event).
			Str("status", input.Status).
			Str("ref_id", input.RefID).
			Msg("event webhook tidak dikenal, diabaikan")
		return nil
	}

	// Step 4: Update status payment di DB
	if err := uc.paymentRepo.UpdateStatus(ctx, existingPayment.ID, newStatus, input.ExternalPaymentID); err != nil {
		return fmt.Errorf("gagal update status payment %s ke %s: %w", existingPayment.ID, newStatus, err)
	}

	// Step 5: Untuk pembayaran sukses, aktifkan subscription
	if newStatus == entity.PaymentPaid {
		if err := uc.activateSubscription(ctx, existingPayment); err != nil {
			return err
		}
	}

	uc.log.Info().
		Str("payment_id", existingPayment.ID.String()).
		Str("ref_id", input.RefID).
		Str("new_status", string(newStatus)).
		Msg("webhook payment berhasil diproses")

	return nil
}

// activateSubscription menangani seluruh alur aktivasi/upgrade subscription setelah payment PAID.
// Ada tiga kasus:
//   1. User belum punya subscription → buat baru + aktivasi (jalur normal)
//   2. User punya subscription ACTIVE dengan plan BERBEDA → upgrade plan (FREE→BASIC/PRO)
//   3. User punya subscription ACTIVE dengan plan SAMA → duplikat webhook, link saja
func (uc *handlePaymentWebhookUseCase) activateSubscription(
	ctx context.Context,
	paidPayment *entity.Payment,
) error {
	durationDays := paidPayment.DurationDays
	if durationDays <= 0 {
		durationDays = subscriptionActivePeriodDays // fallback untuk record lama tanpa duration_days
	}
	periodEnd := time.Now().UTC().AddDate(0, 0, durationDays)

	sub, err := uc.subRepo.CreateSubscription(ctx, paidPayment.UserID, paidPayment.PlanID)
	if err != nil {
		if err != domainerrors.ErrActiveSubscriptionExists {
			return fmt.Errorf("gagal create subscription untuk user %s: %w", paidPayment.UserID, err)
		}

		// User sudah punya subscription ACTIVE — fetch untuk tentukan kasus 2 vs 3
		existingSub, fetchErr := uc.subRepo.GetByUserID(ctx, paidPayment.UserID.String())
		if fetchErr != nil {
			uc.log.Error().
				Err(fetchErr).
				Str("payment_id", paidPayment.ID.String()).
				Str("user_id", paidPayment.UserID.String()).
				Msg("gagal fetch existing active subscription setelah ErrActiveSubscriptionExists")
			return nil
		}

		if existingSub.PlanID == paidPayment.PlanID {
			// Kasus 3: duplikat webhook — plan sama, cukup link payment
			uc.log.Warn().
				Str("payment_id", paidPayment.ID.String()).
				Str("subscription_id", existingSub.ID.String()).
				Msg("payment.success diterima duplikat — link ke existing active subscription")
			if linkErr := uc.paymentRepo.LinkToSubscription(ctx, paidPayment.ID, existingSub.ID); linkErr != nil {
				uc.log.Error().Err(linkErr).
					Str("payment_id", paidPayment.ID.String()).
					Msg("gagal link payment duplikat ke active subscription")
			}
			return nil
		}

		// Kasus 2: upgrade plan (misalnya FREE → BASIC/PRO)
		if upgradeErr := uc.subRepo.UpgradeSubscription(ctx, existingSub.ID, paidPayment.PlanID, periodEnd); upgradeErr != nil {
			return fmt.Errorf("gagal upgrade subscription %s ke plan %s: %w",
				existingSub.ID, paidPayment.PlanID, upgradeErr)
		}

		if linkErr := uc.paymentRepo.LinkToSubscription(ctx, paidPayment.ID, existingSub.ID); linkErr != nil {
			uc.log.Error().
				Err(linkErr).
				Str("payment_id", paidPayment.ID.String()).
				Str("subscription_id", existingSub.ID.String()).
				Msg("gagal link payment ke subscription setelah upgrade — rekonsiliasi manual diperlukan")
		}

		existingSub.PlanID = paidPayment.PlanID // update lokal agar outbox event pakai planID baru
		if outboxErr := uc.publishActivationOutboxEvent(ctx, existingSub, paidPayment.UserID.String(), periodEnd); outboxErr != nil {
			uc.log.Error().
				Err(outboxErr).
				Str("subscription_id", existingSub.ID.String()).
				Msg("gagal insert outbox event subscription.activated setelah upgrade")
		}

		uc.log.Info().
			Str("payment_id", paidPayment.ID.String()).
			Str("subscription_id", existingSub.ID.String()).
			Str("user_id", paidPayment.UserID.String()).
			Str("new_plan_id", paidPayment.PlanID.String()).
			Str("period_end", periodEnd.Format(time.RFC3339)).
			Msg("subscription berhasil diupgrade setelah pembayaran sukses")
		return nil
	}

	// Kasus 1: subscription baru — aktivasi dengan period end
	if err := uc.subRepo.ActivateSubscription(ctx, sub.ID, periodEnd); err != nil {
		return fmt.Errorf("gagal aktivasi subscription %s: %w", sub.ID, err)
	}

	if linkErr := uc.paymentRepo.LinkToSubscription(ctx, paidPayment.ID, sub.ID); linkErr != nil {
		uc.log.Error().
			Err(linkErr).
			Str("payment_id", paidPayment.ID.String()).
			Str("subscription_id", sub.ID.String()).
			Msg("gagal link payment ke subscription — rekonsiliasi manual diperlukan")
	}

	if outboxErr := uc.publishActivationOutboxEvent(ctx, sub, paidPayment.UserID.String(), periodEnd); outboxErr != nil {
		uc.log.Error().
			Err(outboxErr).
			Str("subscription_id", sub.ID.String()).
			Msg("gagal insert outbox event subscription.activated")
	}

	uc.log.Info().
		Str("payment_id", paidPayment.ID.String()).
		Str("subscription_id", sub.ID.String()).
		Str("user_id", paidPayment.UserID.String()).
		Str("period_end", periodEnd.Format(time.RFC3339)).
		Msg("subscription berhasil diaktivasi setelah pembayaran sukses")

	return nil
}

// publishActivationOutboxEvent menyimpan event subscription.activated ke tabel outbox_events.
func (uc *handlePaymentWebhookUseCase) publishActivationOutboxEvent(
	ctx context.Context,
	sub *entity.Subscription,
	userID string,
	periodEnd time.Time,
) error {
	payload := subscriptionActivatedEventPayload{
		SubscriptionID: sub.ID.String(),
		UserID:         userID,
		PlanID:         sub.PlanID.String(),
		PeriodEnd:      periodEnd.Format(time.RFC3339),
		ActivatedAt:    time.Now().UTC().Format(time.RFC3339),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("gagal marshal outbox event payload: %w", err)
	}

	return uc.subRepo.InsertOutboxEvent(
		ctx,
		outboxEventTypeSubscriptionActivated,
		routingKeySubscriptionActivated,
		payloadBytes,
	)
}

// resolvePaymentStatusFromWebhookEvent memetakan event + status string dari orchestrator
// ke PaymentStatus domain. Mengembalikan (status, true) jika event dikenal.
func resolvePaymentStatusFromWebhookEvent(event, statusStr string) (entity.PaymentStatus, bool) {
	switch event {
	case webhookEventPaymentSuccess:
		return entity.PaymentPaid, true
	case webhookEventPaymentFailed:
		return entity.PaymentFailed, true
	case webhookEventPaymentExpired:
		return entity.PaymentExpired, true
	}
	// Fallback: petakan dari field status jika event name tidak dikenal
	switch statusStr {
	case "success":
		return entity.PaymentPaid, true
	case "failed":
		return entity.PaymentFailed, true
	case "expired":
		return entity.PaymentExpired, true
	}
	return "", false
}
