package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/TubagusAldiMY/kasku/billing-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/billing-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/billing-service/internal/domain/repository"
	"github.com/TubagusAldiMY/kasku/billing-service/internal/infrastructure/payment"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// subscriptionPaymentRemarks adalah format teks yang ditampilkan di halaman pembayaran.
const subscriptionPaymentRemarks = "Berlangganan KasKu %s"

// CreateSubscriptionPaymentInput adalah data yang dibutuhkan untuk menginisiasi pembayaran subscription.
type CreateSubscriptionPaymentInput struct {
	UserID        uuid.UUID
	PlanID        uuid.UUID
	PaymentMethod entity.PaymentMethod
	BillingCycle  string // "monthly" (default) atau "yearly"
}

// CreateSubscriptionPaymentOutput adalah data yang dikembalikan ke delivery layer setelah
// payment berhasil diinisiasi di Payment Orchestrator.
type CreateSubscriptionPaymentOutput struct {
	PaymentID  uuid.UUID
	OrderID    string
	AmountIDR  int
	PaymentURL string
	QRString   string     // string QRIS EMV; kosong jika metode bukan QRIS
	ExpiresAt  *time.Time
}

// CreateSubscriptionPaymentUseCase mendefinisikan kontrak untuk inisiasi payment subscription.
//
//go:generate mockgen -source=$GOFILE -destination=../../tests/mocks/mock_create_subscription_payment_usecase.go -package=mocks
type CreateSubscriptionPaymentUseCase interface {
	Execute(ctx context.Context, input CreateSubscriptionPaymentInput) (*CreateSubscriptionPaymentOutput, error)
}

type createSubscriptionPaymentUseCase struct {
	subRepo     repository.SubscriptionRepository
	paymentRepo repository.PaymentRepository
	orchestrator payment.OrchestratorClient
	log         zerolog.Logger
}

// NewCreateSubscriptionPaymentUseCase membuat instance use case inisiasi payment.
func NewCreateSubscriptionPaymentUseCase(
	subRepo repository.SubscriptionRepository,
	paymentRepo repository.PaymentRepository,
	orchestrator payment.OrchestratorClient,
	log zerolog.Logger,
) CreateSubscriptionPaymentUseCase {
	return &createSubscriptionPaymentUseCase{
		subRepo:      subRepo,
		paymentRepo:  paymentRepo,
		orchestrator: orchestrator,
		log:          log,
	}
}

// Execute menginisiasi pembayaran subscription baru:
//  1. Validasi plan dan status subscription user saat ini
//  2. Generate orderID unik
//  3. Call Payment Orchestrator untuk membuat transaksi
//  4. Simpan Payment record ke DB (status PENDING)
//  5. Kembalikan PaymentURL untuk ditampilkan ke user
func (uc *createSubscriptionPaymentUseCase) Execute(
	ctx context.Context,
	input CreateSubscriptionPaymentInput,
) (*CreateSubscriptionPaymentOutput, error) {
	// Step 1: Validasi plan aktif
	plan, err := uc.subRepo.GetPlanWithLimits(ctx, input.PlanID.String())
	if err != nil {
		return nil, fmt.Errorf("validasi plan gagal: %w", err)
	}

	// Hanya plan berbayar yang bisa diproses melalui alur payment ini.
	// Plan FREE (PriceIDR = 0) tidak butuh payment.
	if plan.PriceIDR == 0 {
		return nil, &domainerrors.DomainError{
			Code:    "FREE_PLAN_NO_PAYMENT",
			Message: "Plan FREE tidak memerlukan pembayaran.",
		}
	}

	// Step 2: Cek apakah user sudah punya subscription ACTIVE pada plan BERBAYAR.
	// User dengan FREE subscription diperbolehkan upgrade ke plan berbayar.
	existingSub, subErr := uc.subRepo.GetByUserID(ctx, input.UserID.String())
	if subErr == nil && existingSub.Status == entity.StatusActive {
		// Load plan saat ini untuk cek apakah plan FREE atau berbayar.
		currentPlan, planErr := uc.subRepo.GetPlanWithLimits(ctx, existingSub.PlanID.String())
		if planErr == nil && currentPlan.PriceIDR > 0 {
			// Subscription berbayar aktif — tidak bisa subscribe ulang.
			return nil, domainerrors.ErrActiveSubscriptionExists
		}
		// Plan FREE → lanjutkan sebagai upgrade flow.
	}
	// ErrSubscriptionNotFound adalah kondisi normal (user belum pernah subscribe) — lanjutkan.

	// Step 3: Hitung amount dan durasi berdasarkan billing cycle
	amountIDR := plan.PriceIDR
	durationDays := 30
	if input.BillingCycle == "yearly" {
		amountIDR = plan.PriceIDR * 10 // 10 bulan harga — hemat 2 bulan (≈17%)
		durationDays = 365
	}

	// Step 4: Generate orderID unik sebagai internal reference dan external refId
	orderID := generateOrderID(input.UserID)

	// Step 5: Call Payment Orchestrator — tidak ada retry (idempotency via Idempotency-Key)
	depositReq := payment.DepositRequest{
		RefID:         orderID,
		Amount:        amountIDR,
		Currency:      "IDR",
		PaymentMethod: string(input.PaymentMethod),
		Remarks:       fmt.Sprintf(subscriptionPaymentRemarks, plan.Name),
	}

	depositResp, err := uc.orchestrator.InitiateDeposit(ctx, depositReq, orderID)
	if err != nil {
		uc.log.Error().
			Err(err).
			Str("user_id", input.UserID.String()).
			Str("plan_id", input.PlanID.String()).
			Str("order_id", orderID).
			Msg("payment orchestrator gagal menginisiasi deposit")
		return nil, domainerrors.ErrPaymentGatewayUnavailable
	}

	// Step 6: Simpan Payment record ke DB (status PENDING)
	// PlanID dan DurationDays disimpan di sini agar webhook handler bisa langsung
	// menggunakannya tanpa perlu lookup tabel lain.
	paymentRecord := &entity.Payment{
		ID:                uuid.New(),
		SubscriptionID:    nil, // diisi setelah webhook payment.success diterima
		UserID:            input.UserID,
		PlanID:            input.PlanID,
		OrderID:           orderID,
		AmountIDR:         amountIDR,
		DurationDays:      durationDays,
		Status:            entity.PaymentPending,
		PaymentMethod:     input.PaymentMethod,
		PaymentURL:        depositResp.Data.PaymentURL,
		ExternalPaymentID: depositResp.Data.PaymentID,
		ExternalRefID:     orderID,
		ExpiresAt:         depositResp.Data.ExpiredAt,
	}

	if err := uc.paymentRepo.Create(ctx, paymentRecord); err != nil {
		// Payment sudah dibuat di orchestrator tapi gagal disimpan lokal.
		// Log sebagai error kritis — operator perlu rekonsiliasi manual.
		uc.log.Error().
			Err(err).
			Str("order_id", orderID).
			Str("external_payment_id", depositResp.Data.PaymentID).
			Msg("KRITIS: payment di-create di orchestrator tapi gagal disimpan ke DB lokal")
		return nil, domainerrors.ErrInternal
	}

	uc.log.Info().
		Str("payment_id", paymentRecord.ID.String()).
		Str("order_id", orderID).
		Str("plan_name", plan.Name).
		Int("amount_idr", amountIDR).
		Str("billing_cycle", input.BillingCycle).
		Int("duration_days", durationDays).
		Msg("payment subscription berhasil diinisiasi")

	return &CreateSubscriptionPaymentOutput{
		PaymentID:  paymentRecord.ID,
		OrderID:    orderID,
		AmountIDR:  amountIDR,
		PaymentURL: depositResp.Data.PaymentURL,
		QRString:   depositResp.Data.QRString,
		ExpiresAt:  depositResp.Data.ExpiredAt,
	}, nil
}

// generateOrderID membuat order ID unik dengan format KASKU-SUB-{userID}-{unixTimestamp}.
// Format ini dipilih agar orchestrator bisa trace balik ke user jika diperlukan rekonsiliasi.
func generateOrderID(userID uuid.UUID) string {
	return fmt.Sprintf("KASKU-SUB-%s-%d", userID.String(), time.Now().Unix())
}
