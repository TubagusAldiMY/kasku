package configs

import (
	"fmt"
	"os"
)

type Config struct {
	Server   ServerConfig
	Finance  PostgresConfig
	Billing  PostgresConfig
	User     PostgresConfig
	RabbitMQ RabbitMQConfig
	App      AppConfig
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
		App: AppConfig{
			Env:            getEnvOrDefault("APP_ENV", "development"),
			LogLevel:       getEnvOrDefault("LOG_LEVEL", "info"),
			ServiceVersion: getEnvOrDefault("SERVICE_VERSION", "1.0.0"),
		},
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
