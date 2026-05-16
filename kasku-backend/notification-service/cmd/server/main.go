package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/TubagusAldiMY/kasku/notification-service/configs"
	deliveryhttp "github.com/TubagusAldiMY/kasku/notification-service/internal/delivery/http"
	"github.com/TubagusAldiMY/kasku/notification-service/internal/delivery/http/handler"
	"github.com/TubagusAldiMY/kasku/notification-service/internal/infrastructure/email"
	"github.com/TubagusAldiMY/kasku/notification-service/internal/infrastructure/messaging"
	"github.com/TubagusAldiMY/kasku/notification-service/internal/infrastructure/persistence"
	apptemplates "github.com/TubagusAldiMY/kasku/notification-service/internal/templates"
	"github.com/TubagusAldiMY/kasku/notification-service/internal/usecase"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	httpReadTimeout  = 15 * time.Second
	httpWriteTimeout = 15 * time.Second
	httpIdleTimeout  = 60 * time.Second
	shutdownTimeout  = 30 * time.Second
)

func main() {
	cfg, err := configs.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("gagal load konfigurasi")
	}

	logger := setupLogger(cfg)
	logger.Info().
		Str("service", "notification-service").
		Str("version", cfg.App.ServiceVersion).
		Msg("notification-service starting")

	ctx := context.Background()
	pool, err := persistence.NewPostgresPool(ctx, cfg.Postgres.DSN)
	if err != nil {
		logger.Fatal().Err(err).Msg("gagal koneksi ke PostgreSQL")
	}
	defer pool.Close()
	logger.Info().Msg("PostgreSQL terhubung")

	tmpl, err := apptemplates.LoadEmailTemplates()
	if err != nil {
		logger.Fatal().Err(err).Msg("gagal load HTML templates dari embed")
	}

	var sender email.Sender
	if cfg.IsDevelopment() {
		logger.Info().Msg("mode development: email tidak dikirim (NoOpSender)")
		sender = email.NewNoOpSender()
	} else {
		sender = email.NewSMTPSender(&cfg.SMTP)
	}

	consumer, err := messaging.NewRabbitMQConsumer(cfg.RabbitMQ.URL, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("gagal koneksi ke RabbitMQ")
	}
	defer func() {
		if err := consumer.Close(); err != nil {
			logger.Error().Err(err).Msg("gagal tutup koneksi RabbitMQ")
		}
	}()
	logger.Info().Msg("RabbitMQ terhubung")

	notifUC := usecase.NewNotificationUseCase(sender, tmpl, cfg.App.AppBaseURL, logger)

	consumeCtx, consumeCancel := context.WithCancel(context.Background())
	defer consumeCancel()

	if err := consumer.StartConsuming(consumeCtx, notifUC); err != nil {
		logger.Fatal().Err(err).Msg("gagal mulai consume events")
	}
	logger.Info().Msg("event consumer berjalan")

	preferenceRepo := persistence.NewPostgresPreferenceRepository(pool)
	healthHandler := handler.NewHealthHandler(consumer, pool, cfg.App.ServiceVersion)
	preferenceHandler := handler.NewPreferenceHandler(preferenceRepo)
	router := deliveryhttp.NewRouter(healthHandler, preferenceHandler, cfg.IsDevelopment(), logger)

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  httpReadTimeout,
		WriteTimeout: httpWriteTimeout,
		IdleTimeout:  httpIdleTimeout,
	}

	go func() {
		logger.Info().Str("port", cfg.Server.Port).Msg("notification-service HTTP listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("HTTP server error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	logger.Info().Msg("graceful shutdown dimulai")
	consumeCancel()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("HTTP server shutdown error")
	}
	logger.Info().Msg("notification-service berhenti dengan bersih")
}

func setupLogger(cfg *configs.Config) zerolog.Logger {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	level, err := zerolog.ParseLevel(cfg.App.LogLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)
	return zerolog.New(os.Stdout).With().
		Timestamp().
		Str("service", "notification-service").
		Logger()
}
