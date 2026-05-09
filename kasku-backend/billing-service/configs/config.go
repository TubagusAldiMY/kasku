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
	App      AppConfig
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

// AppConfig menyimpan konfigurasi aplikasi umum.
type AppConfig struct {
	Env                       string
	LogLevel                  string
	ServiceVersion            string
	SubscriptionCheckInterval time.Duration
}

// Load membaca semua konfigurasi dari environment variables.
// Akan panic jika env var wajib tidak ditemukan.
func Load() (*Config, error) {
	checkIntervalMs, err := parseInt64Env("SUBSCRIPTION_CHECK_INTERVAL_MS", 3_600_000) // default 1 jam
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
		App: AppConfig{
			Env:                       getEnvOrDefault("APP_ENV", "development"),
			LogLevel:                  getEnvOrDefault("LOG_LEVEL", "info"),
			ServiceVersion:            getEnvOrDefault("SERVICE_VERSION", "1.0.0"),
			SubscriptionCheckInterval: time.Duration(checkIntervalMs) * time.Millisecond,
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
