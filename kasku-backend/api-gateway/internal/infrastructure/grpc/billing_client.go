package grpc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	billingv1 "github.com/TubagusAldiMY/kasku/api-gateway/proto/billing/v1"
)

// TierLimits menyimpan limit untuk satu user berdasarkan subscription tier-nya.
type TierLimits struct {
	MaxTransactionsPerMonth   int32
	MaxFinancialAccounts      int32
	MaxInvestmentInstruments  int32
	HistoryRetentionMonths    int32
	EmailNotificationsEnabled bool
	ExportCsvEnabled          bool
}

// freeTierLimits adalah default limit untuk tier FREE.
// Digunakan sebagai fallback jika billing-service tidak tersedia.
var freeTierLimits = TierLimits{
	MaxTransactionsPerMonth:   50,
	MaxFinancialAccounts:      3,
	MaxInvestmentInstruments:  0,
	HistoryRetentionMonths:    3,
	EmailNotificationsEnabled: false,
	ExportCsvEnabled:          false,
}

// cacheEntry menyimpan tier limits yang di-cache beserta waktu expiry-nya.
type cacheEntry struct {
	limits    TierLimits
	expiresAt time.Time
}

// BillingClient adalah gRPC client ke billing-service dengan in-memory cache.
type BillingClient struct {
	client   billingv1.BillingInternalClient
	conn     *grpc.ClientConn
	timeout  time.Duration
	cache    map[string]cacheEntry
	cacheMu  sync.RWMutex
	cacheTTL time.Duration
}

// NewBillingClient membuat BillingClient baru dan membuka koneksi gRPC ke billing-service.
func NewBillingClient(addr string, timeout time.Duration) (*BillingClient, error) {
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		return nil, fmt.Errorf("gagal buat gRPC connection ke billing-service: %w", err)
	}

	return &BillingClient{
		client:   billingv1.NewBillingInternalClient(conn),
		conn:     conn,
		timeout:  timeout,
		cache:    make(map[string]cacheEntry),
		cacheTTL: 60 * time.Second,
	}, nil
}

// GetTierLimits mengambil tier limits untuk user_id dari billing-service.
// Jika billing-service tidak tersedia atau timeout, kembalikan FREE tier limits.
// Cache TTL: 60 detik.
func (c *BillingClient) GetTierLimits(ctx context.Context, userID string) TierLimits {
	// Cek cache terlebih dahulu
	c.cacheMu.RLock()
	if entry, ok := c.cache[userID]; ok && time.Now().Before(entry.expiresAt) {
		c.cacheMu.RUnlock()
		return entry.limits
	}
	c.cacheMu.RUnlock()

	// Buat context dengan timeout 300ms untuk billing gRPC call
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.GetUserTierLimits(callCtx, &billingv1.GetUserTierLimitsRequest{
		UserID: userID,
	})
	if err != nil {
		// Fallback ke FREE tier jika billing-service tidak tersedia
		return freeTierLimits
	}

	limits := TierLimits{
		MaxTransactionsPerMonth:   resp.MaxTransactionsPerMonth,
		MaxFinancialAccounts:      resp.MaxFinancialAccounts,
		MaxInvestmentInstruments:  resp.MaxInvestmentInstruments,
		HistoryRetentionMonths:    resp.HistoryRetentionMonths,
		EmailNotificationsEnabled: resp.EmailNotificationsEnabled,
		ExportCsvEnabled:          resp.ExportCsvEnabled,
	}

	// Simpan ke cache
	c.cacheMu.Lock()
	c.cache[userID] = cacheEntry{
		limits:    limits,
		expiresAt: time.Now().Add(c.cacheTTL),
	}
	c.cacheMu.Unlock()

	return limits
}

// Close menutup koneksi gRPC ke billing-service.
func (c *BillingClient) Close() error {
	return c.conn.Close()
}
