package configs

import (
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Config menyimpan semua konfigurasi api-gateway yang dimuat dari environment variables.
type Config struct {
	App     AppConfig
	Server  ServerConfig
	Redis   RedisConfig
	Billing BillingConfig
	JWT     JWTConfig
	CORS    CORSConfig
	Proxy   ProxyConfig
}

type AppConfig struct {
	ServiceVersion string
	Env            string
	LogLevel       string
}

type ServerConfig struct {
	Port string
}

type RedisConfig struct {
	Addr     string
	Password string
}

type BillingConfig struct {
	GRPCAddr string // host:port billing-service gRPC
	Timeout  time.Duration
}

type JWTConfig struct {
	PublicKeyPEM []byte
	PublicKey    *rsa.PublicKey
}

type CORSConfig struct {
	AllowedOrigins []string
}

// ProxyConfig menyimpan upstream URL untuk setiap service.
type ProxyConfig struct {
	AuthServiceURL        string
	UserServiceURL        string
	BillingServiceURL     string
	FinanceServiceURL     string
	TransactionServiceURL string
}

// Load membaca seluruh konfigurasi dari environment variables.
func Load() (*Config, error) {
	cfg := &Config{}

	cfg.App.ServiceVersion = getEnvOrDefault("SERVICE_VERSION", "1.0.0")
	cfg.App.Env = getEnvOrDefault("APP_ENV", "development")
	cfg.App.LogLevel = getEnvOrDefault("LOG_LEVEL", "info")

	cfg.Server.Port = getEnvOrDefault("SERVER_PORT", "8080")

	cfg.Redis.Addr = getEnvOrDefault("REDIS_ADDR", "localhost:6379")
	cfg.Redis.Password = os.Getenv("REDIS_PASSWORD")

	cfg.Billing.GRPCAddr = getEnvOrDefault("BILLING_GRPC_ADDR", "billing-service:9083")
	billingTimeoutMs, _ := strconv.Atoi(getEnvOrDefault("BILLING_GRPC_TIMEOUT_MS", "300"))
	cfg.Billing.Timeout = time.Duration(billingTimeoutMs) * time.Millisecond

	// JWT Public Key — wajib ada untuk verifikasi token
	pubKeyB64 := os.Getenv("JWT_PUBLIC_KEY")
	if pubKeyB64 == "" {
		return nil, fmt.Errorf("JWT_PUBLIC_KEY environment variable wajib diisi")
	}
	pubKeyPEM, err := base64.StdEncoding.DecodeString(pubKeyB64)
	if err != nil {
		return nil, fmt.Errorf("gagal decode JWT_PUBLIC_KEY dari base64: %w", err)
	}
	cfg.JWT.PublicKeyPEM = pubKeyPEM

	pubKey, err := jwt.ParseRSAPublicKeyFromPEM(pubKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("gagal parse RSA public key: %w", err)
	}
	cfg.JWT.PublicKey = pubKey

	// CORS
	originsRaw := getEnvOrDefault("CORS_ALLOWED_ORIGINS", "http://localhost:3000")
	cfg.CORS.AllowedOrigins = strings.Split(originsRaw, ",")
	for i := range cfg.CORS.AllowedOrigins {
		cfg.CORS.AllowedOrigins[i] = strings.TrimSpace(cfg.CORS.AllowedOrigins[i])
	}

	// Upstream service URLs
	cfg.Proxy.AuthServiceURL = getEnvOrDefault("AUTH_SERVICE_URL", "http://auth-service:8081")
	cfg.Proxy.UserServiceURL = getEnvOrDefault("USER_SERVICE_URL", "http://user-service:8082")
	cfg.Proxy.BillingServiceURL = getEnvOrDefault("BILLING_SERVICE_URL", "http://billing-service:8083")
	cfg.Proxy.FinanceServiceURL = getEnvOrDefault("FINANCE_SERVICE_URL", "http://finance-service:8084")
	cfg.Proxy.TransactionServiceURL = getEnvOrDefault("TRANSACTION_SERVICE_URL", "http://transaction-service:8085")

	return cfg, nil
}

// IsDevelopment mengembalikan true jika environment bukan production.
func (c *Config) IsDevelopment() bool {
	return c.App.Env != "production"
}

func getEnvOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
