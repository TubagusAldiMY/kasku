package handler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/TubagusAldiMY/kasku/billing-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/billing-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/billing-service/internal/usecase"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

const (
	// headerUserID adalah nama header yang di-inject oleh api-gateway setelah verifikasi JWT.
	headerUserID = "X-User-ID"

	// headerWebhookSignature adalah nama header signature dari Payment Orchestrator.
	// Orchestrator mengirim signature di header "X-Signature" (lihat webhook.go worker).
	headerWebhookSignature = "X-Signature"
)

// HealthChecker mendefinisikan kontrak untuk health check dependencies.
type HealthChecker interface {
	PingPostgres(ctx context.Context) error
	PingRabbitMQ() error
}

// PlansLister mendefinisikan kontrak use case untuk mengambil daftar plan.
type PlansLister interface {
	Execute(ctx context.Context) ([]entity.SubscriptionPlan, error)
}

// SubscriptionGetter mendefinisikan kontrak use case untuk mengambil subscription user.
type SubscriptionGetter interface {
	Execute(ctx context.Context, userID string) (*entity.Subscription, error)
}

// BillingHandler menangani HTTP request untuk endpoint billing.
// Business logic tidak boleh ada di sini — semua didelegasikan ke use case.
type BillingHandler struct {
	health            HealthChecker
	listPlansUC       PlansLister
	getSubscriptionUC SubscriptionGetter
	createPaymentUC   usecase.CreateSubscriptionPaymentUseCase
	handleWebhookUC   usecase.HandlePaymentWebhookUseCase
	webhookSecret     string
	serviceVersion    string
	log               zerolog.Logger
}

// NewBillingHandler membuat instance BillingHandler baru dengan semua dependensi yang diinjeksikan.
func NewBillingHandler(
	health HealthChecker,
	listPlansUC PlansLister,
	getSubscriptionUC SubscriptionGetter,
	createPaymentUC usecase.CreateSubscriptionPaymentUseCase,
	handleWebhookUC usecase.HandlePaymentWebhookUseCase,
	webhookSecret string,
	serviceVersion string,
	log zerolog.Logger,
) *BillingHandler {
	return &BillingHandler{
		health:            health,
		listPlansUC:       listPlansUC,
		getSubscriptionUC: getSubscriptionUC,
		createPaymentUC:   createPaymentUC,
		handleWebhookUC:   handleWebhookUC,
		webhookSecret:     webhookSecret,
		serviceVersion:    serviceVersion,
		log:               log,
	}
}

// Health mengembalikan status kesehatan service beserta status masing-masing dependency.
// HTTP 200 = healthy, HTTP 503 = degraded (salah satu dependency bermasalah).
func (h *BillingHandler) Health(c *gin.Context) {
	ctx := c.Request.Context()
	status := "healthy"
	httpStatus := http.StatusOK

	checks := gin.H{}
	if err := h.health.PingPostgres(ctx); err != nil {
		checks["postgres"] = "unhealthy"
		status = "degraded"
		httpStatus = http.StatusServiceUnavailable
	} else {
		checks["postgres"] = "healthy"
	}
	if err := h.health.PingRabbitMQ(); err != nil {
		checks["rabbitmq"] = "unhealthy"
		status = "degraded"
		httpStatus = http.StatusServiceUnavailable
	} else {
		checks["rabbitmq"] = "healthy"
	}

	c.JSON(httpStatus, gin.H{
		"status":  status,
		"version": h.serviceVersion,
		"checks":  checks,
	})
}

// planResponse adalah DTO untuk respons daftar plan — memisahkan domain entity dari transport layer.
type planResponse struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	PriceIDR int               `json:"price_idr"`
	Limits   entity.PlanLimits `json:"limits"`
}

// ListPlans mengembalikan semua subscription plan yang aktif.
// Endpoint ini public (tidak memerlukan autentikasi) untuk keperluan halaman pricing.
func (h *BillingHandler) ListPlans(c *gin.Context) {
	plans, err := h.listPlansUC.Execute(c.Request.Context())
	if err != nil {
		h.log.Error().Err(err).Msg("gagal mengambil daftar subscription plan")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   gin.H{"code": "INTERNAL_ERROR", "message": "Gagal mengambil daftar plan."},
		})
		return
	}

	result := make([]planResponse, 0, len(plans))
	for _, p := range plans {
		result = append(result, planResponse{
			ID:       p.ID.String(),
			Name:     p.Name,
			PriceIDR: p.PriceIDR,
			Limits:   p.Limits,
		})
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

// GetSubscription mengembalikan detail subscription aktif milik user yang terautentikasi.
// Memerlukan header X-User-ID yang di-inject oleh api-gateway.
func (h *BillingHandler) GetSubscription(c *gin.Context) {
	userID := c.GetHeader(headerUserID)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   gin.H{"code": "UNAUTHORIZED", "message": "Header autentikasi tidak ditemukan."},
		})
		return
	}

	sub, err := h.getSubscriptionUC.Execute(c.Request.Context(), userID)
	if err != nil {
		if domainerrors.IsDomainError(err) {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   gin.H{"code": "SUBSCRIPTION_NOT_FOUND", "message": "Subscription tidak ditemukan."},
			})
			return
		}
		correlationID, _ := c.Get("correlation_id")
		h.log.Error().
			Err(err).
			Str("user_id", userID).
			Interface("correlation_id", correlationID).
			Msg("gagal mengambil subscription")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   gin.H{"code": "INTERNAL_ERROR", "message": "Gagal mengambil subscription."},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"id":                   sub.ID.String(),
			"user_id":              sub.UserID.String(),
			"plan_id":              sub.PlanID.String(),
			"status":               string(sub.Status),
			"current_period_start": sub.CurrentPeriodStart,
			"current_period_end":   sub.CurrentPeriodEnd,
		},
	})
}

// subscribeRequest adalah DTO request untuk endpoint inisiasi pembayaran subscription.
type subscribeRequest struct {
	PlanID        string `json:"plan_id" binding:"required"`
	PaymentMethod string `json:"payment_method"` // default: "QRIS" jika kosong
	BillingCycle  string `json:"billing_cycle"`  // "monthly" (default) atau "yearly"
}

// subscribeResponse adalah DTO response berisi informasi pembayaran untuk ditampilkan ke user.
type subscribeResponse struct {
	PaymentID  string  `json:"payment_id"`
	OrderID    string  `json:"order_id"`
	AmountIDR  int     `json:"amount_idr"`
	PaymentURL string  `json:"payment_url"`
	QRString   string  `json:"qr_string,omitempty"` // string QRIS EMV; hanya ada jika metode QRIS
	ExpiresAt  *string `json:"expires_at,omitempty"`
}

// Subscribe menginisiasi pembayaran subscription baru.
// Memerlukan header X-User-ID (di-inject api-gateway).
// Mengembalikan payment_url yang harus ditampilkan ke user untuk menyelesaikan pembayaran.
func (h *BillingHandler) Subscribe(c *gin.Context) {
	userID := c.GetHeader(headerUserID)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   gin.H{"code": "UNAUTHORIZED", "message": "Header autentikasi tidak ditemukan."},
		})
		return
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_USER_ID", "message": "Format user ID tidak valid."},
		})
		return
	}

	var req subscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_REQUEST", "message": "plan_id wajib diisi."},
		})
		return
	}

	planUUID, err := uuid.Parse(req.PlanID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_PLAN_ID", "message": "Format plan_id tidak valid."},
		})
		return
	}

	paymentMethod, isValid := entity.ParsePaymentMethod(req.PaymentMethod)
	if !isValid {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_PAYMENT_METHOD",
				"message": "Metode pembayaran tidak valid. Gunakan QRIS atau VIRTUAL_ACCOUNT.",
			},
		})
		return
	}

	output, err := h.createPaymentUC.Execute(c.Request.Context(), usecase.CreateSubscriptionPaymentInput{
		UserID:        userUUID,
		PlanID:        planUUID,
		PaymentMethod: paymentMethod,
		BillingCycle:  req.BillingCycle,
	})
	if err != nil {
		h.mapPaymentErrorToResponse(c, err, userID)
		return
	}

	resp := subscribeResponse{
		PaymentID:  output.PaymentID.String(),
		OrderID:    output.OrderID,
		AmountIDR:  output.AmountIDR,
		PaymentURL: output.PaymentURL,
		QRString:   output.QRString,
	}
	if output.ExpiresAt != nil {
		formatted := output.ExpiresAt.Format("2006-01-02T15:04:05Z07:00")
		resp.ExpiresAt = &formatted
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": resp})
}

// webhookPayload adalah DTO untuk parsing body webhook dari Payment Orchestrator.
// Field names mengikuti format snake_case yang dikirim oleh orchestrator.
type webhookPayload struct {
	Type      string  `json:"type"`             // "deposit" — kategori transaksi
	Status    string  `json:"status"`           // "success" | "failed" | "expired"
	PaymentID string  `json:"provider_trx_id"`  // ID transaksi dari provider payment
	RefID     string  `json:"ref_id"`           // ref_id yang kita kirim = internal OrderID
	Amount    int     `json:"amount"`
	PaidAt    *string `json:"paidAt,omitempty"` // opsional
}

// PaymentWebhook menerima notifikasi event dari Payment Orchestrator.
// Endpoint ini tidak memerlukan JWT (tidak ada header X-User-ID).
// Keamanan dijamin melalui verifikasi HMAC-SHA256 signature.
//
// CATATAN: Endpoint ini selalu mengembalikan HTTP 200 untuk event yang dikenal,
// bahkan jika terjadi idempotency skip. Ini mencegah orchestrator melakukan retry
// yang tidak perlu karena logic error di sisi kita.
func (h *BillingHandler) PaymentWebhook(c *gin.Context) {
	// Step 1: Baca raw body SEBELUM binding JSON — raw body diperlukan untuk verifikasi HMAC
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.log.Error().Err(err).Msg("gagal membaca body webhook")
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_REQUEST", "message": "Gagal membaca request body."},
		})
		return
	}

	// Step 2: Verifikasi HMAC-SHA256 signature SEBELUM memproses apapun
	incomingSignature := c.GetHeader(headerWebhookSignature)
	if !h.isValidWebhookSignature(rawBody, incomingSignature) {
		// Log tanpa menyertakan signature asli untuk keamanan
		correlationID, _ := c.Get("correlation_id")
		h.log.Warn().
			Interface("correlation_id", correlationID).
			Str("remote_addr", c.RemoteIP()).
			Msg("webhook ditolak: signature tidak valid")
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_SIGNATURE", "message": "Signature webhook tidak valid."},
		})
		return
	}

	// Step 3: Parse JSON dari raw body yang sudah terverifikasi
	var payload webhookPayload
	if err := bindJSONFromBytes(rawBody, &payload); err != nil {
		h.log.Error().Err(err).Msg("gagal parse JSON webhook yang sudah terverifikasi")
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "INVALID_JSON", "message": "Format JSON tidak valid."},
		})
		return
	}

	// Step 4: Validasi field wajib — ref_id diperlukan untuk lookup payment di DB
	if payload.RefID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   gin.H{"code": "MISSING_FIELDS", "message": "Field ref_id wajib ada."},
		})
		return
	}

	// Step 5: Parse PaidAt jika ada
	// Orchestrator menggunakan status ("success"/"failed"/"expired") sebagai event key.
	input := usecase.PaymentWebhookInput{
		Event:             payload.Status,
		ExternalPaymentID: payload.PaymentID,
		RefID:             payload.RefID,
		Amount:            payload.Amount,
		Status:            payload.Status,
	}
	if payload.PaidAt != nil {
		parsed, parseErr := parseRFC3339Time(*payload.PaidAt)
		if parseErr == nil {
			input.PaidAt = &parsed
		}
	}

	// Step 6: Delegasikan ke use case — tidak ada business logic di sini
	if err := h.handleWebhookUC.Execute(c.Request.Context(), input); err != nil {
		correlationID, _ := c.Get("correlation_id")
		h.log.Error().
			Err(err).
			Str("ref_id", payload.RefID).
			Str("status", payload.Status).
			Interface("correlation_id", correlationID).
			Msg("gagal memproses webhook payment")
		// Return 500 agar orchestrator retry — ini adalah kegagalan server kita, bukan input invalid
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   gin.H{"code": "PROCESSING_ERROR", "message": "Gagal memproses notifikasi payment."},
		})
		return
	}

	// Selalu 200 untuk event yang berhasil diproses (termasuk idempotency skip)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// isValidWebhookSignature memverifikasi HMAC-SHA256 signature webhook.
// HMAC dihitung: HMAC-SHA256(webhookSecret, rawBody) dan hasilnya di-hex encode.
// Menggunakan hmac.Equal untuk constant-time comparison (cegah timing attack).
func (h *BillingHandler) isValidWebhookSignature(rawBody []byte, incomingSignature string) bool {
	if incomingSignature == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(h.webhookSecret))
	mac.Write(rawBody)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(incomingSignature), []byte(expectedSignature))
}

// mapPaymentErrorToResponse memetakan domain error dari use case ke HTTP response yang tepat.
// Menggunakan errors.As agar error yang di-wrap oleh fmt.Errorf tetap dapat dideteksi.
func (h *BillingHandler) mapPaymentErrorToResponse(c *gin.Context, err error, userID string) {
	var domainErr *domainerrors.DomainError
	if !errors.As(err, &domainErr) {
		correlationID, _ := c.Get("correlation_id")
		h.log.Error().
			Err(err).
			Str("user_id", userID).
			Interface("correlation_id", correlationID).
			Msg("error tidak dikenal dari create payment use case")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   gin.H{"code": "INTERNAL_ERROR", "message": "Terjadi kesalahan internal."},
		})
		return
	}

	httpStatus := domainErrorToHTTPStatus(domainErr.Code)
	c.JSON(httpStatus, gin.H{
		"success": false,
		"error":   gin.H{"code": domainErr.Code, "message": domainErr.Message},
	})
}

// domainErrorToHTTPStatus memetakan domain error code ke HTTP status code yang semantik.
func domainErrorToHTTPStatus(errorCode string) int {
	switch errorCode {
	case "PLAN_NOT_FOUND":
		return http.StatusNotFound
	case "ACTIVE_SUBSCRIPTION_EXISTS":
		return http.StatusConflict
	case "INVALID_PAYMENT_METHOD":
		return http.StatusBadRequest
	case "FREE_PLAN_NO_PAYMENT":
		return http.StatusUnprocessableEntity
	case "PAYMENT_GATEWAY_UNAVAILABLE":
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// bindJSONFromBytes mem-parse JSON dari byte slice ke target struct.
// Digunakan setelah raw body dibaca untuk signature verification.
func bindJSONFromBytes(data []byte, target *webhookPayload) error {
	return json.Unmarshal(data, target)
}

// parseRFC3339Time mem-parse string waktu dalam format RFC 3339.
func parseRFC3339Time(raw string) (time.Time, error) {
	return time.Parse(time.RFC3339, raw)
}
