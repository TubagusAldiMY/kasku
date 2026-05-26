package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	obsmetrics "github.com/TubagusAldiMY/kasku/observability-go/metrics"
	"github.com/TubagusAldiMY/kasku/transaction-service/configs"
	deliveryhttp "github.com/TubagusAldiMY/kasku/transaction-service/internal/delivery/http"
	"github.com/TubagusAldiMY/kasku/transaction-service/internal/delivery/http/handler"
	grpcserver "github.com/TubagusAldiMY/kasku/transaction-service/internal/infrastructure/grpc"
	"github.com/TubagusAldiMY/kasku/transaction-service/internal/infrastructure/persistence"
	"github.com/TubagusAldiMY/kasku/transaction-service/internal/usecase"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace/noop"
)

const (
	gracefulShutdownTimeout = 30 * time.Second
	httpReadTimeout         = 15 * time.Second
	httpWriteTimeout        = 15 * time.Second
	httpIdleTimeout         = 60 * time.Second
)

func main() {
	cfg, err := configs.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("gagal load konfigurasi")
	}

	logger := buildLogger(cfg)
	logger.Info().
		Str("service", "transaction-service").
		Str("version", cfg.App.ServiceVersion).
		Msg("transaction-service starting")

	// Inisialisasi OpenTelemetry distributed tracing.
	// Jika OTEL_EXPORTER_OTLP_ENDPOINT kosong, noop tracer dipasang — service tetap jalan normal.
	otelShutdown := initTracer(cfg, logger)
	defer otelShutdown()

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

	txHandler := buildHandler(pool, cfg, logger)
	metricsReg := obsmetrics.NewRegistry("transaction-service")
	metricsReg.RegisterDBPool(pool)
	router := deliveryhttp.NewRouter(txHandler, cfg.IsDevelopment(), metricsReg, logger)

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  httpReadTimeout,
		WriteTimeout: httpWriteTimeout,
		IdleTimeout:  httpIdleTimeout,
	}

	grpcSrv := grpcserver.NewTransactionGRPCServer(pool, logger)
	if err := grpcSrv.Start(cfg.Server.GRPCPort); err != nil {
		logger.Fatal().Err(err).Msg("gagal start transaction gRPC server")
	}

	go func() {
		logger.Info().Str("port", cfg.Server.Port).Msg("transaction-service listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("HTTP server error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	logger.Info().Msg("graceful shutdown dimulai")
	grpcSrv.Stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), gracefulShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("HTTP server shutdown error")
	}
	logger.Info().Msg("transaction-service berhenti dengan bersih")
}

func buildLogger(cfg *configs.Config) zerolog.Logger {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	level, err := zerolog.ParseLevel(cfg.App.LogLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)
	return zerolog.New(os.Stdout).With().Timestamp().Str("service", "transaction-service").Logger()
}

func buildHandler(pool *pgxpool.Pool, cfg *configs.Config, logger zerolog.Logger) *handler.TransactionHandler {
	txRepo := persistence.NewPostgresTransactionRepository(pool)
	catRepo := persistence.NewPostgresCategoryRepository(pool)
	budgetRepo := persistence.NewPostgresBudgetRepository(pool)

	return handler.NewTransactionHandler(
		usecase.NewCreateTransactionUseCase(txRepo, catRepo),
		usecase.NewListTransactionsUseCase(txRepo),
		usecase.NewGetTransactionUseCase(txRepo),
		usecase.NewUpdateTransactionUseCase(txRepo),
		usecase.NewDeleteTransactionUseCase(txRepo),
		usecase.NewExportCSVUseCase(txRepo),
		usecase.NewListCategoriesUseCase(catRepo),
		usecase.NewCreateCategoryUseCase(catRepo),
		usecase.NewUpdateCategoryUseCase(catRepo),
		usecase.NewDeleteCategoryUseCase(catRepo),
		usecase.NewCreateBudgetUseCase(budgetRepo),
		usecase.NewListBudgetsUseCase(budgetRepo),
		usecase.NewGetBudgetUseCase(budgetRepo),
		usecase.NewUpdateBudgetUseCase(budgetRepo),
		usecase.NewDeleteBudgetUseCase(budgetRepo),
		&appHealthChecker{pool: pool},
		cfg.App.ServiceVersion,
		logger,
	)
}

// appHealthChecker mengimplementasikan handler.HealthChecker menggunakan pgxpool.
type appHealthChecker struct{ pool *pgxpool.Pool }

func (h *appHealthChecker) PingPostgres(ctx context.Context) error {
	return persistence.PingPostgres(ctx, h.pool)
}

// initTracer menginisialisasi OpenTelemetry tracer provider global.
// Mengembalikan fungsi shutdown yang harus dipanggil saat graceful shutdown.
// Jika OTELEndpoint kosong, noop tracer dipasang — tidak pernah crash.
func initTracer(cfg *configs.Config, logger zerolog.Logger) func() {
	noopShutdown := func() {}
	setNoop := func() {
		otel.SetTracerProvider(noop.NewTracerProvider())
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		))
	}

	if cfg.OTEL.Endpoint == "" {
		setNoop()
		return noopShutdown
	}

	otelCtx := context.Background()
	exp, err := otlptracehttp.New(otelCtx,
		otlptracehttp.WithEndpoint(cfg.OTEL.Endpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		logger.Warn().Err(err).Msg("OTel exporter init gagal, tracing dinonaktifkan")
		setNoop()
		return noopShutdown
	}

	res, err := resource.New(otelCtx,
		resource.WithAttributes(
			semconv.ServiceName("transaction-service"),
			semconv.ServiceVersion(cfg.App.ServiceVersion),
			semconv.DeploymentEnvironment(cfg.App.Env),
		),
	)
	if err != nil {
		res = resource.Default()
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp,
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithMaxExportBatchSize(512),
		),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	logger.Info().Str("endpoint", cfg.OTEL.Endpoint).Msg("OTel tracing aktif")

	return func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tp.Shutdown(shutdownCtx); err != nil {
			logger.Error().Err(err).Msg("OTel tracer provider shutdown error")
		}
	}
}
