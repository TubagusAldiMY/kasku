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
	App        AppConfig
}

type ServerConfig struct {
	Port string
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

	return &Config{
		Server: ServerConfig{
			Port: getEnvOrDefault("SERVER_PORT", "8081"),
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
