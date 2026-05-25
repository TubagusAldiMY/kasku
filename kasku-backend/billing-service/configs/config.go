package configs

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config menyimpan seluruh konfigurasi billing-service yang di-load dari environment variables.
type Config struct {
	Server   ServerConfig
	Postgres PostgresConfig
	RabbitMQ RabbitMQConfig
	App      AppConfig
	Cleanup  CleanupConfig
	Outbox   OutboxConfig
	Payment  PaymentConfig
}

// ServerConfig menyimpan konfigurasi port HTTP dan gRPC.
type ServerConfig struct {
	Port     string
	GRPCPort string
}

// PostgresConfig menyimpan DSN koneksi PostgreSQL.
type PostgresConfig struct {
	DSN string
}

// RabbitMQConfig menyimpan URL AMQP untuk publisher event.
type RabbitMQConfig struct {
	URL string
}

// AppConfig menyimpan konfigurasi aplikasi umum.
type AppConfig struct {
	Env                         string
	LogLevel                    string
	ServiceVersion              string
	OTELEndpoint                string
	SubscriptionCheckInterval   time.Duration
	ExpiringNotificationEnabled bool
}

// CleanupConfig mengatur retention/garbage-collection job.
type CleanupConfig struct {
	Interval time.Duration
	DryRun   bool
}

// OutboxConfig mengatur outbox dispatcher.
type OutboxConfig struct {
	PollInterval time.Duration
	BatchSize    int
}

// PaymentConfig menyimpan konfigurasi integrasi Payment Orchestrator.
// Semua nilai WAJIB diisi dari environment variables — tidak ada nilai default untuk security reason.
type PaymentConfig struct {
	// OrchestratorBaseURL adalah base URL Payment Orchestrator (contoh: https://api-payment.roemahprogram.com)
	OrchestratorBaseURL string

	// OrchestratorAPIKey adalah Bearer token untuk autentikasi ke Payment Orchestrator.
	// Gunakan sk_live_... untuk production, sk_test_... untuk staging/dev.
	OrchestratorAPIKey string

	// WebhookSecret adalah secret untuk verifikasi HMAC-SHA256 signature webhook.
	WebhookSecret string

	// CallbackBaseURL adalah base URL service KasKu yang dapat diakses oleh orchestrator
	// untuk mengirimkan webhook callback. Contoh: https://api.kasku.app
	CallbackBaseURL string
}

// Load membaca semua konfigurasi dari environment variables.
// Akan panic jika env var wajib tidak ditemukan.
func Load() (*Config, error) {
	checkIntervalMs, err := parseInt64Env("SUBSCRIPTION_CHECK_INTERVAL_MS", 3_600_000) // default 1 jam
	if err != nil {
		return nil, err
	}

	cleanupIntervalMs, err := parseInt64Env("CLEANUP_INTERVAL_MS", 3_600_000) // default 1 jam
	if err != nil {
		return nil, err
	}

	outboxPollMs, err := parseInt64Env("OUTBOX_POLL_INTERVAL_MS", 2_000) // default 2 detik
	if err != nil {
		return nil, err
	}

	outboxBatchSize, err := parseInt64Env("OUTBOX_BATCH_SIZE", 25)
	if err != nil {
		return nil, err
	}

	cleanupDryRun, err := parseBoolEnv("CLEANUP_DRY_RUN", false)
	if err != nil {
		return nil, err
	}

	expiringEnabled, err := parseBoolEnv("BILLING_EXPIRING_NOTIFICATION_ENABLED", false)
	if err != nil {
		return nil, err
	}

	return &Config{
		Server: ServerConfig{
			Port:     getEnvOrDefault("SERVER_PORT", "8083"),
			GRPCPort: getEnvOrDefault("GRPC_PORT", "9083"),
		},
		Postgres: PostgresConfig{
			DSN: requireEnv("POSTGRES_DSN"),
		},
		RabbitMQ: RabbitMQConfig{
			URL: requireEnv("RABBITMQ_URL"),
		},
		App: AppConfig{
			Env:                         getEnvOrDefault("APP_ENV", "development"),
			LogLevel:                    getEnvOrDefault("LOG_LEVEL", "info"),
			ServiceVersion:              getEnvOrDefault("SERVICE_VERSION", "1.0.0"),
			OTELEndpoint:                os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
			SubscriptionCheckInterval:   time.Duration(checkIntervalMs) * time.Millisecond,
			ExpiringNotificationEnabled: expiringEnabled,
		},
		Cleanup: CleanupConfig{
			Interval: time.Duration(cleanupIntervalMs) * time.Millisecond,
			DryRun:   cleanupDryRun,
		},
		Outbox: OutboxConfig{
			PollInterval: time.Duration(outboxPollMs) * time.Millisecond,
			BatchSize:    int(outboxBatchSize),
		},
		Payment: PaymentConfig{
			OrchestratorBaseURL: requireEnv("PAYMENT_ORCHESTRATOR_BASE_URL"),
			OrchestratorAPIKey:  requireEnv("PAYMENT_ORCHESTRATOR_API_KEY"),
			WebhookSecret:       requireEnv("PAYMENT_WEBHOOK_SECRET"),
			CallbackBaseURL:     requireEnv("PAYMENT_CALLBACK_BASE_URL"),
		},
	}, nil
}

// IsDevelopment mengembalikan true jika service berjalan di mode development.
func (c *Config) IsDevelopment() bool {
	return c.App.Env == "development"
}

// requireEnv membaca env var wajib — panic jika kosong agar startup gagal cepat.
func requireEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		panic(fmt.Sprintf("environment variable wajib tidak ditemukan: %s", key))
	}
	return val
}

// getEnvOrDefault membaca env var dengan nilai default jika kosong.
func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

// parseInt64Env mem-parse env var sebagai int64, mengembalikan defaultValue jika tidak diset.
func parseInt64Env(key string, defaultValue int64) (int64, error) {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue, nil
	}
	parsed, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s harus berupa angka valid: %w", key, err)
	}
	return parsed, nil
}

// parseBoolEnv mem-parse env var sebagai boolean dengan toleransi "true/false/1/0/yes/no" (case-insensitive).
func parseBoolEnv(key string, defaultValue bool) (bool, error) {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue, nil
	}
	switch val {
	case "1", "true", "TRUE", "True", "yes", "YES", "y", "Y":
		return true, nil
	case "0", "false", "FALSE", "False", "no", "NO", "n", "N":
		return false, nil
	}
	return false, fmt.Errorf("%s tidak valid (gunakan true/false/1/0/yes/no): %q", key, val)
}
