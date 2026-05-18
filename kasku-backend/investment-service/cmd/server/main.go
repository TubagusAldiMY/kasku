package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/TubagusAldiMY/kasku/investment-service/configs"
	deliveryhttp "github.com/TubagusAldiMY/kasku/investment-service/internal/delivery/http"
	"github.com/TubagusAldiMY/kasku/investment-service/internal/delivery/http/handler"
	grpcserver "github.com/TubagusAldiMY/kasku/investment-service/internal/infrastructure/grpc"
	"github.com/TubagusAldiMY/kasku/investment-service/internal/infrastructure/persistence"
	priceinfra "github.com/TubagusAldiMY/kasku/investment-service/internal/infrastructure/price"
	"github.com/TubagusAldiMY/kasku/investment-service/internal/usecase"
	obsmetrics "github.com/TubagusAldiMY/kasku/observability-go/metrics"
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
		Str("service", "investment-service").
		Str("version", cfg.App.ServiceVersion).
		Str("env", cfg.App.Env).
		Msg("investment-service starting")

	// Database — investment-service tidak punya migration sendiri. Tabel
	// investment_assets & unit_history di tiap tenant schema dibuat oleh
	// stored function public.provision_tenant() yang dimiliki finance-service.
	// Lihat finance-service/migrations/000003_extend_provision_tenant_with_investment.up.sql
	ctx := context.Background()
	pool, err := persistence.NewPostgresPool(ctx, cfg.Postgres.DSN)
	if err != nil {
		logger.Fatal().Err(err).Msg("gagal koneksi ke PostgreSQL")
	}
	defer pool.Close()
	logger.Info().Msg("PostgreSQL terhubung")

	// Dependency injection: wiring semua layer
	assetRepo := persistence.NewPostgresInvestmentRepository(pool)
	priceClient := priceinfra.NewHTTPClient(cfg.Price.HTTPURL, cfg.Price.Timeout)

	createUC := usecase.NewCreateAssetUseCase(assetRepo)
	listUC := usecase.NewListAssetsUseCase(assetRepo, priceClient)
	getUC := usecase.NewGetAssetUseCase(assetRepo)
	updateUC := usecase.NewUpdateAssetUseCase(assetRepo)
	deleteUC := usecase.NewDeleteAssetUseCase(assetRepo)
	recordUC := usecase.NewRecordUnitChangeUseCase(assetRepo)
	historyUC := usecase.NewGetUnitHistoryUseCase(assetRepo)

	healthChecker := &postgresHealthChecker{pool: pool}
	investmentHandler := handler.NewInvestmentHandler(
		createUC, listUC, getUC, updateUC, deleteUC, recordUC, historyUC,
		healthChecker, cfg.App.ServiceVersion, logger,
	)

	metricsReg := obsmetrics.NewRegistry("investment-service")
	metricsReg.RegisterDBPool(pool)
	router := deliveryhttp.NewRouter(investmentHandler, cfg.IsDevelopment(), metricsReg)

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	grpcSrv := grpcserver.NewInvestmentGRPCServer(pool, logger)
	if err := grpcSrv.Start(cfg.Server.GRPCPort); err != nil {
		logger.Fatal().Err(err).Msg("gagal start investment gRPC server")
	}

	go func() {
		logger.Info().Str("port", cfg.Server.Port).Msg("investment-service HTTP server listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("HTTP server error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	logger.Info().Msg("graceful shutdown dimulai (timeout: 30s)")
	grpcSrv.Stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), gracefulShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("HTTP server shutdown error")
	}
	logger.Info().Msg("investment-service berhenti dengan bersih")
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
		Str("service", "investment-service").
		Logger()
}

type postgresHealthChecker struct {
	pool *pgxpool.Pool
}

func (h *postgresHealthChecker) PingPostgres(ctx context.Context) error {
	return persistence.PingPostgres(ctx, h.pool)
}
