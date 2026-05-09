package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/TubagusAldiMY/kasku/billing-service/configs"
	deliveryhttp "github.com/TubagusAldiMY/kasku/billing-service/internal/delivery/http"
	"github.com/TubagusAldiMY/kasku/billing-service/internal/delivery/http/handler"
	"github.com/TubagusAldiMY/kasku/billing-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/billing-service/internal/domain/repository"
	billinggrpc "github.com/TubagusAldiMY/kasku/billing-service/internal/infrastructure/grpc"
	"github.com/TubagusAldiMY/kasku/billing-service/internal/infrastructure/persistence"
	"github.com/TubagusAldiMY/kasku/billing-service/internal/usecase"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

const gracefulShutdownTimeout = 30 * time.Second

func main() {
	cfg, err := configs.Load()
	if err != nil {
		startupLog := zerolog.New(os.Stdout).With().Timestamp().Logger()
		startupLog.Fatal().Err(err).Msg("gagal load konfigurasi")
	}

	logger := buildLogger(cfg)
	logger.Info().
		Str("service", "billing-service").
		Str("version", cfg.App.ServiceVersion).
		Str("env", cfg.App.Env).
		Msg("billing-service starting")

	// Jalankan migration sebelum menerima traffic apapun
	logger.Info().Msg("menjalankan database migrations")
	if err := persistence.RunMigrations(cfg.Postgres.DSN); err != nil {
		logger.Fatal().Err(err).Msg("gagal menjalankan database migrations")
	}
	logger.Info().Msg("database migrations selesai")

	ctx := context.Background()

	// Buat connection pool PostgreSQL
	pool, err := persistence.NewPostgresPool(ctx, cfg.Postgres.DSN)
	if err != nil {
		logger.Fatal().Err(err).Msg("gagal membuat koneksi ke PostgreSQL")
	}
	defer pool.Close()
	logger.Info().Msg("PostgreSQL terhubung")

	// Wiring dependency injection: repository → use case → handler/server
	subRepo := persistence.NewPostgresSubscriptionRepository(pool)

	getTierLimitsUC := usecase.NewGetTierLimitsUseCase(subRepo)
	listPlansUC := usecase.NewListPlansUseCase(subRepo)
	getSubscriptionUC := usecase.NewGetSubscriptionUseCase(subRepo)

	// Mulai gRPC server pada port 9083
	grpcServer := billinggrpc.NewBillingGRPCServer(getTierLimitsUC, logger)
	if err := grpcServer.Start(cfg.Server.GRPCPort); err != nil {
		logger.Fatal().Err(err).Msg("gagal memulai gRPC server")
	}
	defer grpcServer.Stop()
	logger.Info().Str("port", cfg.Server.GRPCPort).Msg("gRPC server berjalan")

	// Siapkan HTTP server
	healthChecker := &postgresHealthChecker{pool: pool}
	billingHandler := handler.NewBillingHandler(
		healthChecker,
		listPlansUC,
		getSubscriptionUC,
		cfg.App.ServiceVersion,
		logger,
	)
	router := deliveryhttp.NewRouter(billingHandler, cfg.IsDevelopment(), logger)

	httpServer := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Mulai HTTP server di goroutine terpisah
	go func() {
		logger.Info().Str("port", cfg.Server.Port).Msg("billing-service HTTP listening")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("HTTP server berhenti dengan error")
		}
	}()

	// Mulai background job untuk update expired subscriptions
	go runSubscriptionExpiryCheck(ctx, subRepo, cfg.App.SubscriptionCheckInterval, logger)

	// Tunggu sinyal shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	logger.Info().Msg("menerima sinyal shutdown, memulai graceful shutdown (30s timeout)")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), gracefulShutdownTimeout)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("HTTP server shutdown tidak bersih")
	}

	logger.Info().Msg("billing-service berhenti dengan bersih")
}

// buildLogger membuat zerolog.Logger dengan format dan level yang sesuai konfigurasi.
func buildLogger(cfg *configs.Config) zerolog.Logger {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	level, err := zerolog.ParseLevel(cfg.App.LogLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)
	return zerolog.New(os.Stdout).With().
		Timestamp().
		Str("service", "billing-service").
		Logger()
}

// postgresHealthChecker mengadaptasi pgxpool.Pool ke interface HealthChecker yang dibutuhkan handler.
type postgresHealthChecker struct {
	pool *pgxpool.Pool
}

func (h *postgresHealthChecker) PingPostgres(ctx context.Context) error {
	return persistence.PingPostgres(ctx, h.pool)
}

// runSubscriptionExpiryCheck adalah background job yang berjalan periodik untuk mengubah status
// subscription yang sudah melewati current_period_end menjadi EXPIRED.
// Interval dikonfigurasi via SUBSCRIPTION_CHECK_INTERVAL_MS (default 1 jam).
func runSubscriptionExpiryCheck(
	ctx context.Context,
	subRepo repository.SubscriptionRepository,
	interval time.Duration,
	log zerolog.Logger,
) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("subscription expiry check job berhenti")
			return
		case <-ticker.C:
			expired, err := subRepo.ListExpiredSubscriptions(ctx)
			if err != nil {
				log.Error().Err(err).Msg("gagal mengambil daftar expired subscriptions")
				continue
			}

			for _, sub := range expired {
				if err := subRepo.UpdateStatus(ctx, sub.ID.String(), entity.StatusExpired); err != nil {
					log.Error().
						Err(err).
						Str("subscription_id", sub.ID.String()).
						Msg("gagal update status subscription ke EXPIRED")
				}
			}

			if len(expired) > 0 {
				log.Info().Int("count", len(expired)).Msg("subscription expired berhasil diupdate")
			}
		}
	}
}
