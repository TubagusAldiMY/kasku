package price

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type HTTPClient struct {
	baseURL string
	client  *http.Client
}

type priceResponse struct {
	Success bool `json:"success"`
	Data    *struct {
		PriceIDR float64 `json:"price_idr"`
		IsFresh  bool    `json:"is_fresh"`
	} `json:"data"`
}

func NewHTTPClient(baseURL string, timeout time.Duration) *HTTPClient {
	return &HTTPClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: timeout},
	}
}

func (c *HTTPClient) GetPrice(ctx context.Context, symbol string) (float64, bool, error) {
	escapedSymbol := url.PathEscape(symbol)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/prices/"+escapedSymbol, nil)
	if err != nil {
		return 0, false, fmt.Errorf("gagal membuat request price-service: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, false, fmt.Errorf("gagal call price-service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, false, fmt.Errorf("price-service return status %d", resp.StatusCode)
	}

	var out priceResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return 0, false, fmt.Errorf("gagal decode response price-service: %w", err)
	}
	if !out.Success || out.Data == nil {
		return 0, false, fmt.Errorf("price-service response tidak sukses")
	}

	return out.Data.PriceIDR, out.Data.IsFresh, nil
}
