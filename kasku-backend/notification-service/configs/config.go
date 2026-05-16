package configs

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Server   ServerConfig
	Postgres PostgresConfig
	RabbitMQ RabbitMQConfig
	SMTP     SMTPConfig
	App      AppConfig
}

type ServerConfig struct {
	Port string
}

type RabbitMQConfig struct {
	URL string
}

type PostgresConfig struct {
	DSN string
}

type SMTPConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
}

type AppConfig struct {
	Env            string
	LogLevel       string
	ServiceVersion string
	AppBaseURL     string
}

func Load() (*Config, error) {
	smtpPort, err := strconv.Atoi(getEnvOrDefault("SMTP_PORT", "587"))
	if err != nil {
		return nil, fmt.Errorf("SMTP_PORT harus berupa angka: %w", err)
	}

	return &Config{
		Server: ServerConfig{
			Port: getEnvOrDefault("SERVER_PORT", "8089"),
		},
		Postgres: PostgresConfig{
			DSN: requireEnv("POSTGRES_DSN"),
		},
		RabbitMQ: RabbitMQConfig{
			URL: requireEnv("RABBITMQ_URL"),
		},
		SMTP: SMTPConfig{
			Host:     getEnvOrDefault("SMTP_HOST", "localhost"),
			Port:     smtpPort,
			User:     os.Getenv("SMTP_USER"),
			Password: os.Getenv("SMTP_PASS"),
			From:     getEnvOrDefault("SMTP_FROM", "noreply@kasku.app"),
		},
		App: AppConfig{
			Env:            getEnvOrDefault("APP_ENV", "development"),
			LogLevel:       getEnvOrDefault("LOG_LEVEL", "info"),
			ServiceVersion: getEnvOrDefault("SERVICE_VERSION", "1.0.0"),
			AppBaseURL:     getEnvOrDefault("APP_BASE_URL", "http://localhost:3000"),
		},
	}, nil
}

func (c *Config) IsDevelopment() bool { return c.App.Env == "development" }

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
