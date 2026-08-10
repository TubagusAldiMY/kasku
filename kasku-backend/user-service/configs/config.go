package configs

import (
	"fmt"
	"os"
)

type Config struct {
	Server       ServerConfig
	Finance      PostgresConfig
	Billing      PostgresConfig
	User         PostgresConfig
	RabbitMQ     RabbitMQConfig
	AuthService  AuthServiceConfig
	App          AppConfig
	OTELEndpoint string
}

type ServerConfig struct {
	Port     string
	GRPCPort string
}

type PostgresConfig struct {
	DSN string
}

type RabbitMQConfig struct {
	URL string
}

// AuthServiceConfig menyimpan alamat gRPC auth-service beserta shared secret
// untuk RPC sensitif (UpdateUsername).
type AuthServiceConfig struct {
	GRPCAddr       string
	InternalSecret string
}

type AppConfig struct {
	Env            string
	LogLevel       string
	ServiceVersion string
}

func Load() (*Config, error) {
	return &Config{
		Server: ServerConfig{
			Port:     getEnvOrDefault("SERVER_PORT", "8082"),
			GRPCPort: getEnvOrDefault("GRPC_PORT", "9082"),
		},
		Finance: PostgresConfig{
			DSN: requireEnv("POSTGRES_FINANCE_DSN"),
		},
		Billing: PostgresConfig{
			DSN: requireEnv("POSTGRES_BILLING_DSN"),
		},
		User: PostgresConfig{
			DSN: requireEnv("POSTGRES_USER_DSN"),
		},
		RabbitMQ: RabbitMQConfig{
			URL: requireEnv("RABBITMQ_URL"),
		},
		AuthService: AuthServiceConfig{
			GRPCAddr: getEnvOrDefault("AUTH_GRPC_ADDR", "auth-service:9081"),
			// Sengaja TIDAK requireEnv: kalau kosong, auth-service hanya mencatat
			// warning dan tetap melayani (mode dev). Memaksa wajib di sini akan
			// membuat user-service gagal boot di lingkungan dev yang selama ini jalan.
			InternalSecret: os.Getenv("INTERNAL_GRPC_SECRET"),
		},
		App: AppConfig{
			Env:            getEnvOrDefault("APP_ENV", "development"),
			LogLevel:       getEnvOrDefault("LOG_LEVEL", "info"),
			ServiceVersion: getEnvOrDefault("SERVICE_VERSION", "1.0.0"),
		},
		OTELEndpoint: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
	}, nil
}

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
