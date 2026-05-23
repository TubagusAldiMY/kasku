package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	orchestratorRequestTimeout = 10 * time.Second

	// depositEndpointPath adalah path endpoint inisiasi pembayaran di Payment Orchestrator.
	depositEndpointPath = "/v1/payment/deposit"

	// statusEndpointPath adalah path endpoint cek status pembayaran. {refId} di-replace saat runtime.
	statusEndpointPath = "/v1/payment/status/"
)

// DepositRequest adalah body request untuk menginisiasi pembayaran ke Payment Orchestrator.
type DepositRequest struct {
	RefID         string `json:"refId"`
	Amount        int    `json:"amount"`
	Currency      string `json:"currency"`
	PaymentMethod string `json:"paymentMethod"`
	Remarks       string `json:"remarks"`
}

// DepositResponseData adalah field data dari respons sukses inisiasi deposit.
// Orchestrator mengembalikan snake_case: provider_trx_id, payment_url, expires_at.
type DepositResponseData struct {
	PaymentID  string     `json:"provider_trx_id"` // ID unik dari provider/orchestrator
	RefID      string     `json:"ref_id"`
	Status     string     `json:"status"`
	Amount     int        `json:"amount"`
	PaymentURL string     `json:"payment_url"`
	QRString   string     `json:"qr_string"`
	ExpiredAt  *time.Time `json:"expires_at"`
}

// DepositResponse adalah seluruh respons dari endpoint POST /v1/payment/deposit.
type DepositResponse struct {
	Success bool                `json:"success"`
	Data    DepositResponseData `json:"data"`
}

// StatusResponseData adalah field data dari respons cek status pembayaran.
type StatusResponseData struct {
	PaymentID string     `json:"provider_trx_id"`
	RefID     string     `json:"ref_id"`
	Status    string     `json:"status"`
	Amount    int        `json:"amount"`
	PaidAt    *time.Time `json:"paid_at"`
}

// StatusResponse adalah seluruh respons dari endpoint GET /v1/payment/status/{refId}.
type StatusResponse struct {
	Success bool               `json:"success"`
	Data    StatusResponseData `json:"data"`
}

// OrchestratorClient mendefinisikan kontrak untuk komunikasi dengan Payment Orchestrator.
// Dipisahkan sebagai interface agar dapat di-mock pada unit test use case.
//
//go:generate mockgen -source=$GOFILE -destination=../../../../tests/mocks/mock_orchestrator_client.go -package=mocks
type OrchestratorClient interface {
	InitiateDeposit(ctx context.Context, req DepositRequest, idempotencyKey string) (*DepositResponse, error)
	CheckStatus(ctx context.Context, refID string) (*StatusResponse, error)
}

// httpOrchestratorClient adalah implementasi OrchestratorClient yang berkomunikasi
// dengan Payment Orchestrator via HTTP + Bearer token.
type httpOrchestratorClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewHTTPOrchestratorClient membuat instance client Payment Orchestrator baru.
// baseURL: https://api-payment.roemahprogram.com
// apiKey: sk_live_... atau sk_test_...
func NewHTTPOrchestratorClient(baseURL, apiKey string) OrchestratorClient {
	return &httpOrchestratorClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: orchestratorRequestTimeout,
		},
	}
}

// InitiateDeposit menginisiasi pembayaran baru ke Payment Orchestrator.
// idempotencyKey digunakan sebagai Idempotency-Key header untuk mencegah double charge
// jika terjadi retry karena network issue.
func (c *httpOrchestratorClient) InitiateDeposit(
	ctx context.Context,
	req DepositRequest,
	idempotencyKey string,
) (*DepositResponse, error) {
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("gagal marshal deposit request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+depositEndpointPath,
		bytes.NewReader(bodyBytes),
	)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat HTTP request deposit: %w", err)
	}

	c.applyCommonHeaders(httpReq)
	httpReq.Header.Set("Idempotency-Key", idempotencyKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("gagal menghubungi payment orchestrator: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gagal membaca response body dari orchestrator: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("payment orchestrator mengembalikan status %d: %s",
			resp.StatusCode, truncateForLog(rawBody, 200))
	}

	var depositResp DepositResponse
	if err := json.Unmarshal(rawBody, &depositResp); err != nil {
		return nil, fmt.Errorf("gagal unmarshal deposit response dari orchestrator: %w", err)
	}

	if !depositResp.Success {
		return nil, fmt.Errorf("payment orchestrator melaporkan kegagalan inisiasi deposit")
	}

	return &depositResp, nil
}

// CheckStatus mengecek status pembayaran berdasarkan refId yang sebelumnya kita kirimkan.
func (c *httpOrchestratorClient) CheckStatus(ctx context.Context, refID string) (*StatusResponse, error) {
	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL+statusEndpointPath+refID,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat HTTP request cek status: %w", err)
	}

	c.applyCommonHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("gagal cek status ke payment orchestrator: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gagal membaca response body status: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("payment orchestrator status check mengembalikan HTTP %d", resp.StatusCode)
	}

	var statusResp StatusResponse
	if err := json.Unmarshal(rawBody, &statusResp); err != nil {
		return nil, fmt.Errorf("gagal unmarshal status response: %w", err)
	}

	return &statusResp, nil
}

// applyCommonHeaders menambahkan header Authorization dan Content-Type ke setiap request.
func (c *httpOrchestratorClient) applyCommonHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
}

// truncateForLog memotong byte slice untuk logging agar tidak mencatat data sensitif yang panjang.
func truncateForLog(data []byte, maxLen int) string {
	if len(data) <= maxLen {
		return string(data)
	}
	return string(data[:maxLen]) + "...[truncated]"
}
