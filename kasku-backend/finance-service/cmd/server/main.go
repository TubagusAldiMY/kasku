package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/TubagusAldiMY/kasku/finance-service/configs"
	deliveryhttp "github.com/TubagusAldiMY/kasku/finance-service/internal/delivery/http"
	"github.com/TubagusAldiMY/kasku/finance-service/internal/delivery/http/handler"
	"github.com/TubagusAldiMY/kasku/finance-service/internal/infrastructure/persistence"
	"github.com/TubagusAldiMY/kasku/finance-service/internal/usecase"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const gracefulShutdownTimeout = 30 * time.Second

func main() {
	cfg, err := configs.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("gagal load konfigurasi")
	}

	logger := buildLogger(cfg)
	logger.Info().
		Str("service", "finance-service").
		Str("version", cfg.App.ServiceVersion).
		Str("env", cfg.App.Env).
		Msg("finance-service starting")

	logger.Info().Msg("menjalankan database migrations")
	if err := persistence.RunMigrations(cfg.Postgres.DSN); err != nil {
		logger.Fatal().Err(err).Msg("gagal menjalankan migrations")
	}

	ctx := context.Background()
	pool, err := persistence.NewPostgresPool(ctx, cfg.Postgres.DSN)
	if err != nil {
		logger.Fatal().Err(err).Msg("gagal koneksi ke PostgreSQL")
	}
	defer pool.Close()
	logger.Info().Msg("PostgreSQL terhubung")

	// Dependency injection: wiring semua layer
	accountRepo := persistence.NewPostgresAccountRepository(pool)

	createUC := usecase.NewCreateAccountUseCase(accountRepo)
	listUC := usecase.NewListAccountsUseCase(accountRepo)
	getUC := usecase.NewGetAccountUseCase(accountRepo)
	updateUC := usecase.NewUpdateAccountUseCase(accountRepo)
	deleteUC := usecase.NewDeleteAccountUseCase(accountRepo)
	historyUC := usecase.NewGetBalanceHistoryUseCase(accountRepo)

	healthChecker := &postgresHealthChecker{pool: pool}
	accountHandler := handler.NewAccountHandler(
		createUC, listUC, getUC, updateUC, deleteUC, historyUC,
		healthChecker, cfg.App.ServiceVersion, logger,
	)

	router := deliveryhttp.NewRouter(accountHandler, cfg.IsDevelopment())

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info().Str("port", cfg.Server.Port).Msg("finance-service HTTP server listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("HTTP server error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	logger.Info().Msg("graceful shutdown dimulai (timeout: 30s)")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), gracefulShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("HTTP server shutdown error")
	}
	logger.Info().Msg("finance-service berhenti dengan bersih")
}

func buildLogger(cfg *configs.Config) zerolog.Logger {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	level, err := zerolog.ParseLevel(cfg.App.LogLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)
	return zerolog.New(os.Stdout).With().
		Timestamp().
		Str("service", "finance-service").
		Logger()
}

// postgresHealthChecker mengimplementasikan handler.HealthChecker untuk dependency injection.
type postgresHealthChecker struct {
	pool *pgxpool.Pool
}

func (h *postgresHealthChecker) PingPostgres(ctx context.Context) error {
	return persistence.PingPostgres(ctx, h.pool)
}
