package configs

import "os"

type Config struct {
	Server ServerConfig
	App    AppConfig
}

type ServerConfig struct {
	Port string
}

type AppConfig struct {
	Env            string
	LogLevel       string
	ServiceVersion string
}

func Load() (*Config, error) {
	return &Config{
		Server: ServerConfig{
			Port: getEnvOrDefault("SERVER_PORT", "8090"),
		},
		App: AppConfig{
			Env:            getEnvOrDefault("APP_ENV", "development"),
			LogLevel:       getEnvOrDefault("LOG_LEVEL", "info"),
			ServiceVersion: getEnvOrDefault("SERVICE_VERSION", "0.1.0"),
		},
	}, nil
}

func (c *Config) IsDevelopment() bool { return c.App.Env == "development" }

func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}
