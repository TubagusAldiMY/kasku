package configs

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config menggabungkan semua subkonfigurasi yang dibaca dari env vars.
type Config struct {
	Server    ServerConfig
	App       AppConfig
	Postgres  PostgresConfig
	Redis     RedisConfig
	JWT       AdminJWTConfig
	Argon2    Argon2Config
	Bootstrap BootstrapAdminConfig
	OTEL      OTELConfig
}

// OTELConfig menyimpan konfigurasi OpenTelemetry exporter.
type OTELConfig struct {
	// Endpoint OTLP gRPC collector, contoh: "kasku-otel-collector:4317".
	// Kosongkan untuk menonaktifkan tracing (noop provider).
	Endpoint string
}

// ServerConfig adalah pengaturan HTTP server.
type ServerConfig struct {
	Port string
}

// AppConfig adalah identitas + level logging.
type AppConfig struct {
	Env            string
	LogLevel       string
	ServiceVersion string
}

// PostgresConfig membawa 3 DSN: kasku_admin (R/W), kasku_auth (R+limited write),
// kasku_billing (R+limited write). admin-service membonceng credential service
// pemilik untuk operasi cross-DB (lihat README untuk trade-off-nya).
type PostgresConfig struct {
	AdminDSN   string
	AuthDSN    string
	BillingDSN string
}

// RedisConfig untuk JWT blacklist admin.
type RedisConfig struct {
	Addr     string
	Password string
}

// AdminJWTConfig menyimpan secret + TTL untuk admin HS256 JWT.
type AdminJWTConfig struct {
	Secret string
	TTL    time.Duration
}

// Argon2Config sama dengan auth-service.
type Argon2Config struct {
	Time      uint32
	MemoryKB  uint32
	Threads   uint8
	KeyLength uint32
}

// BootstrapAdminConfig dipakai saat tabel admin_users kosong.
type BootstrapAdminConfig struct {
	Username string
	Password string
}

// Load membaca env vars dan menyusun Config. Mengembalikan error pada nilai
// yang tidak valid; meminta env wajib yang absen via requireEnv (panik).
func Load() (*Config, error) {
	adminTTL, err := time.ParseDuration(getEnvOrDefault("ADMIN_JWT_TTL", "8h"))
	if err != nil {
		return nil, fmt.Errorf("ADMIN_JWT_TTL tidak valid: %w", err)
	}

	argon2Time, err := parseUint32Env("ARGON2_TIME", 3)
	if err != nil {
		return nil, err
	}
	argon2Memory, err := parseUint32Env("ARGON2_MEMORY_KB", 65536)
	if err != nil {
		return nil, err
	}
	argon2Threads, err := parseUint8Env("ARGON2_THREADS", 4)
	if err != nil {
		return nil, err
	}
	argon2KeyLen, err := parseUint32Env("ARGON2_KEY_LENGTH", 32)
	if err != nil {
		return nil, err
	}

	return &Config{
		Server: ServerConfig{
			Port: getEnvOrDefault("SERVER_PORT", "8090"),
		},
		App: AppConfig{
			Env:            getEnvOrDefault("APP_ENV", "development"),
			LogLevel:       getEnvOrDefault("LOG_LEVEL", "info"),
			ServiceVersion: getEnvOrDefault("SERVICE_VERSION", "1.0.0"),
		},
		Postgres: PostgresConfig{
			AdminDSN:   requireEnv("POSTGRES_ADMIN_DSN"),
			AuthDSN:    requireEnv("POSTGRES_AUTH_ADMIN_DSN"),
			BillingDSN: requireEnv("POSTGRES_BILLING_ADMIN_DSN"),
		},
		Redis: RedisConfig{
			Addr:     getEnvOrDefault("REDIS_ADDR", "redis:6379"),
			Password: os.Getenv("REDIS_PASSWORD"),
		},
		JWT: AdminJWTConfig{
			Secret: requireEnv("ADMIN_JWT_SECRET"),
			TTL:    adminTTL,
		},
		Argon2: Argon2Config{
			Time:      argon2Time,
			MemoryKB:  argon2Memory,
			Threads:   argon2Threads,
			KeyLength: argon2KeyLen,
		},
		Bootstrap: BootstrapAdminConfig{
			Username: os.Getenv("ADMIN_BOOTSTRAP_USERNAME"),
			Password: os.Getenv("ADMIN_BOOTSTRAP_PASSWORD"),
		},
		OTEL: OTELConfig{
			Endpoint: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		},
	}, nil
}

// IsDevelopment mengembalikan true bila env mode development.
func (c *Config) IsDevelopment() bool {
	return c.App.Env == "development"
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("environment variable wajib tidak ditemukan: %s", key))
	}
	return v
}

func getEnvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func parseUint32Env(key string, def uint32) (uint32, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	n, err := strconv.ParseUint(v, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%s harus angka positif: %w", key, err)
	}
	return uint32(n), nil
}

func parseUint8Env(key string, def uint8) (uint8, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	n, err := strconv.ParseUint(v, 10, 8)
	if err != nil {
		return 0, fmt.Errorf("%s harus angka 0-255: %w", key, err)
	}
	return uint8(n), nil
}
