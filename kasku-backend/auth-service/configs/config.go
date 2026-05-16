package configs

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config merupakan representasi seluruh konfigurasi aplikasi.
// Semua nilai berasal dari environment variables (12-factor app).
type Config struct {
	Server     ServerConfig
	Postgres   PostgresConfig
	JWT        JWTConfig
	Argon2     Argon2Config
	BruteForce BruteForceConfig
	Redis      RedisConfig
	RabbitMQ   RabbitMQConfig
	RateLimit  RateLimitConfig
	Cleanup    CleanupConfig
	App        AppConfig
}

type ServerConfig struct {
	Port           string
	GRPCPort       string
	InternalSecret string // shared secret untuk gRPC RPC sensitif (mis. RevokeUserTokens)
}

type PostgresConfig struct {
	DSN string
}

type JWTConfig struct {
	PrivateKeyPEM   []byte
	PublicKeyPEM    []byte
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

type Argon2Config struct {
	Time      uint32
	MemoryKB  uint32
	Threads   uint8
	KeyLength uint32
}

type BruteForceConfig struct {
	MaxAttempts     int16
	LockoutDuration time.Duration
}

type RedisConfig struct {
	Addr     string
	Password string
}

type RabbitMQConfig struct {
	URL string
}

type RateLimitConfig struct {
	Enabled           bool
	RegisterPerWindow int
	LoginPerWindow    int
	ForgotIPPerWindow int
	ForgotEmailLimit  int
	ResendEmailLimit  int
	IPWindow          time.Duration // window untuk per-IP limits (mis. 1m)
	EmailWindow       time.Duration // window untuk per-email limits (mis. 1h)
}

type CleanupConfig struct {
	Enabled  bool
	Interval time.Duration
	DryRun   bool
}

type AppConfig struct {
	Env            string
	LogLevel       string
	ServiceVersion string
}

// Load membaca semua environment variables dan mengembalikan Config tervalidasi.
// Mengembalikan error jika ada variable wajib yang tidak ada atau nilai tidak valid.
func Load() (*Config, error) {
	// JWT keys: base64 StdEncoding dari PEM bytes
	privKeyBase64 := requireEnv("JWT_PRIVATE_KEY")
	privKeyPEM, err := base64.StdEncoding.DecodeString(privKeyBase64)
	if err != nil {
		return nil, fmt.Errorf("gagal decode JWT_PRIVATE_KEY dari base64: %w", err)
	}

	pubKeyBase64 := requireEnv("JWT_PUBLIC_KEY")
	pubKeyPEM, err := base64.StdEncoding.DecodeString(pubKeyBase64)
	if err != nil {
		return nil, fmt.Errorf("gagal decode JWT_PUBLIC_KEY dari base64: %w", err)
	}

	accessTTL, err := time.ParseDuration(getEnvOrDefault("JWT_ACCESS_TOKEN_TTL", "15m"))
	if err != nil {
		return nil, fmt.Errorf("JWT_ACCESS_TOKEN_TTL tidak valid: %w", err)
	}

	refreshTTL, err := time.ParseDuration(getEnvOrDefault("JWT_REFRESH_TOKEN_TTL", "720h"))
	if err != nil {
		return nil, fmt.Errorf("JWT_REFRESH_TOKEN_TTL tidak valid: %w", err)
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

	maxAttempts, err := parseInt16Env("BRUTE_FORCE_MAX_ATTEMPTS", 5)
	if err != nil {
		return nil, err
	}

	lockoutDurationStr := getEnvOrDefault("BRUTE_FORCE_LOCKOUT_DURATION", "15m")
	lockoutDuration, err := time.ParseDuration(lockoutDurationStr)
	if err != nil {
		return nil, fmt.Errorf("BRUTE_FORCE_LOCKOUT_DURATION tidak valid: %w", err)
	}

	rateLimitEnabled := parseBoolEnv("RATE_LIMIT_ENABLED", true)
	registerPerWin, err := parseIntEnv("RATE_LIMIT_REGISTER_PER_MIN", 10)
	if err != nil {
		return nil, err
	}
	loginPerWin, err := parseIntEnv("RATE_LIMIT_LOGIN_PER_MIN", 10)
	if err != nil {
		return nil, err
	}
	forgotIPPerWin, err := parseIntEnv("RATE_LIMIT_FORGOT_IP_PER_MIN", 5)
	if err != nil {
		return nil, err
	}
	forgotEmailLimit, err := parseIntEnv("RATE_LIMIT_FORGOT_EMAIL_PER_HOUR", 3)
	if err != nil {
		return nil, err
	}
	resendEmailLimit, err := parseIntEnv("RATE_LIMIT_RESEND_EMAIL_PER_HOUR", 3)
	if err != nil {
		return nil, err
	}
	ipWindow, err := time.ParseDuration(getEnvOrDefault("RATE_LIMIT_IP_WINDOW", "1m"))
	if err != nil {
		return nil, fmt.Errorf("RATE_LIMIT_IP_WINDOW tidak valid: %w", err)
	}
	emailWindow, err := time.ParseDuration(getEnvOrDefault("RATE_LIMIT_EMAIL_WINDOW", "1h"))
	if err != nil {
		return nil, fmt.Errorf("RATE_LIMIT_EMAIL_WINDOW tidak valid: %w", err)
	}

	cleanupEnabled := parseBoolEnv("CLEANUP_ENABLED", true)
	cleanupInterval, err := time.ParseDuration(getEnvOrDefault("CLEANUP_INTERVAL", "1h"))
	if err != nil {
		return nil, fmt.Errorf("CLEANUP_INTERVAL tidak valid: %w", err)
	}
	cleanupDryRun := parseBoolEnv("CLEANUP_DRY_RUN", false)

	return &Config{
		Server: ServerConfig{
			Port:           getEnvOrDefault("SERVER_PORT", "8081"),
			GRPCPort:       getEnvOrDefault("GRPC_PORT", "9081"),
			InternalSecret: os.Getenv("INTERNAL_GRPC_SECRET"),
		},
		Postgres: PostgresConfig{
			DSN: requireEnv("POSTGRES_DSN"),
		},
		JWT: JWTConfig{
			PrivateKeyPEM:   privKeyPEM,
			PublicKeyPEM:    pubKeyPEM,
			AccessTokenTTL:  accessTTL,
			RefreshTokenTTL: refreshTTL,
		},
		Argon2: Argon2Config{
			Time:      argon2Time,
			MemoryKB:  argon2Memory,
			Threads:   argon2Threads,
			KeyLength: argon2KeyLen,
		},
		BruteForce: BruteForceConfig{
			MaxAttempts:     maxAttempts,
			LockoutDuration: lockoutDuration,
		},
		Redis: RedisConfig{
			Addr:     getEnvOrDefault("REDIS_ADDR", "redis:6379"),
			Password: os.Getenv("REDIS_PASSWORD"),
		},
		RabbitMQ: RabbitMQConfig{
			URL: requireEnv("RABBITMQ_URL"),
		},
		RateLimit: RateLimitConfig{
			Enabled:           rateLimitEnabled,
			RegisterPerWindow: registerPerWin,
			LoginPerWindow:    loginPerWin,
			ForgotIPPerWindow: forgotIPPerWin,
			ForgotEmailLimit:  forgotEmailLimit,
			ResendEmailLimit:  resendEmailLimit,
			IPWindow:          ipWindow,
			EmailWindow:       emailWindow,
		},
		Cleanup: CleanupConfig{
			Enabled:  cleanupEnabled,
			Interval: cleanupInterval,
			DryRun:   cleanupDryRun,
		},
		App: AppConfig{
			Env:            getEnvOrDefault("APP_ENV", "development"),
			LogLevel:       getEnvOrDefault("LOG_LEVEL", "info"),
			ServiceVersion: getEnvOrDefault("SERVICE_VERSION", "1.0.0"),
		},
	}, nil
}

// IsDevelopment mengembalikan true jika aplikasi berjalan di mode development.
func (c *Config) IsDevelopment() bool {
	return c.App.Env == "development"
}

func requireEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		panic(fmt.Sprintf("environment variable wajib tidak ditemukan: %s", key))
	}
	return val
}

func getEnvOrDefault(key, defaultValue string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	return val
}

func parseUint32Env(key string, defaultValue uint32) (uint32, error) {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue, nil
	}
	parsed, err := strconv.ParseUint(val, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%s harus berupa angka positif: %w", key, err)
	}
	return uint32(parsed), nil
}

func parseUint8Env(key string, defaultValue uint8) (uint8, error) {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue, nil
	}
	parsed, err := strconv.ParseUint(val, 10, 8)
	if err != nil {
		return 0, fmt.Errorf("%s harus berupa angka 0-255: %w", key, err)
	}
	return uint8(parsed), nil
}

func parseIntEnv(key string, defaultValue int) (int, error) {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue, nil
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return 0, fmt.Errorf("%s harus berupa angka: %w", key, err)
	}
	return parsed, nil
}

func parseBoolEnv(key string, defaultValue bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}
	switch val {
	case "1", "true", "TRUE", "True", "yes", "YES":
		return true
	case "0", "false", "FALSE", "False", "no", "NO":
		return false
	default:
		return defaultValue
	}
}

func parseInt16Env(key string, defaultValue int16) (int16, error) {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue, nil
	}
	parsed, err := strconv.ParseInt(val, 10, 16)
	if err != nil {
		return 0, fmt.Errorf("%s harus berupa angka: %w", key, err)
	}
	return int16(parsed), nil
}

// ErrMissingEnv digunakan saat environment variable wajib tidak ada.
var ErrMissingEnv = errors.New("environment variable wajib tidak ditemukan")
