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
	grpcserver "github.com/TubagusAldiMY/kasku/finance-service/internal/infrastructure/grpc"
	"github.com/TubagusAldiMY/kasku/finance-service/internal/infrastructure/persistence"
	"github.com/TubagusAldiMY/kasku/finance-service/internal/usecase"
	obsmetrics "github.com/TubagusAldiMY/kasku/observability-go/metrics"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace/noop"
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

	// ── OpenTelemetry tracing init ────────────────────────────────────────────
	if cfg.App.OTELEndpoint == "" {
		otel.SetTracerProvider(noop.NewTracerProvider())
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		))
		logger.Info().Msg("OTel endpoint tidak dikonfigurasi, tracing dinonaktifkan (noop)")
	} else {
		otelCtx := context.Background()
		exp, err := otlptracegrpc.New(otelCtx,
			otlptracegrpc.WithEndpoint(cfg.App.OTELEndpoint),
			otlptracegrpc.WithInsecure(),
		)
		if err != nil {
			logger.Warn().Err(err).Msg("OTel exporter init gagal, tracing dinonaktifkan")
			otel.SetTracerProvider(noop.NewTracerProvider())
			otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
				propagation.TraceContext{},
				propagation.Baggage{},
			))
		} else {
			res, _ := resource.New(otelCtx,
				resource.WithAttributes(
					semconv.ServiceName("finance-service"),
					semconv.ServiceVersion(cfg.App.ServiceVersion),
					semconv.DeploymentEnvironment(cfg.App.Env),
				),
			)
			tp := sdktrace.NewTracerProvider(
				sdktrace.WithBatcher(exp),
				sdktrace.WithResource(res),
				sdktrace.WithSampler(sdktrace.AlwaysSample()),
			)
			otel.SetTracerProvider(tp)
			otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
				propagation.TraceContext{},
				propagation.Baggage{},
			))
			defer func() {
				ctx2, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = tp.Shutdown(ctx2)
			}()
			logger.Info().Str("endpoint", cfg.App.OTELEndpoint).Msg("OTel tracing aktif")
		}
	}
	// ─────────────────────────────────────────────────────────────────────────

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

	metricsReg := obsmetrics.NewRegistry("finance-service")
	metricsReg.RegisterDBPool(pool)
	router := deliveryhttp.NewRouter(accountHandler, cfg.IsDevelopment(), metricsReg)

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	grpcSrv := grpcserver.NewFinanceGRPCServer(pool, logger)
	if err := grpcSrv.Start(cfg.Server.GRPCPort); err != nil {
		logger.Fatal().Err(err).Msg("gagal start finance gRPC server")
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
	grpcSrv.Stop()

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
