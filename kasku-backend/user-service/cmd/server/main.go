package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	obsmetrics "github.com/TubagusAldiMY/kasku/observability-go/metrics"
	"github.com/TubagusAldiMY/kasku/user-service/configs"
	deliveryhttp "github.com/TubagusAldiMY/kasku/user-service/internal/delivery/http"
	"github.com/TubagusAldiMY/kasku/user-service/internal/delivery/http/handler"
	"github.com/TubagusAldiMY/kasku/user-service/internal/infrastructure/messaging"
	"github.com/TubagusAldiMY/kasku/user-service/internal/infrastructure/persistence"
	"github.com/TubagusAldiMY/kasku/user-service/internal/usecase"
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

func main() {
	cfg, err := configs.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("gagal load konfigurasi")
	}

	logger := setupLogger(cfg)
	logger.Info().
		Str("service", "user-service").
		Str("version", cfg.App.ServiceVersion).
		Str("env", cfg.App.Env).
		Msg("user-service starting")

	ctx := context.Background()

	// ── OTel Tracing ─────────────────────────────────────────────────────
	if cfg.OTELEndpoint == "" {
		otel.SetTracerProvider(noop.NewTracerProvider())
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{}, propagation.Baggage{}),
		)
		logger.Info().Msg("OTel endpoint tidak dikonfigurasi, tracing dinonaktifkan (noop)")
	} else {
		otelCtx := context.Background()
		exp, err := otlptracegrpc.New(otelCtx,
			otlptracegrpc.WithEndpoint(cfg.OTELEndpoint),
			otlptracegrpc.WithInsecure(),
		)
		if err != nil {
			logger.Warn().Err(err).Msg("OTel exporter init gagal, tracing dinonaktifkan")
			otel.SetTracerProvider(noop.NewTracerProvider())
			otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
				propagation.TraceContext{}, propagation.Baggage{}))
		} else {
			res, _ := resource.New(otelCtx, resource.WithAttributes(
				semconv.ServiceName("user-service"),
				semconv.ServiceVersion(cfg.App.ServiceVersion),
				semconv.DeploymentEnvironment(cfg.App.Env),
			))
			tp := sdktrace.NewTracerProvider(
				sdktrace.WithBatcher(exp),
				sdktrace.WithResource(res),
				sdktrace.WithSampler(sdktrace.AlwaysSample()),
			)
			otel.SetTracerProvider(tp)
			otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
				propagation.TraceContext{}, propagation.Baggage{}))
			defer func() {
				ctx2, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = tp.Shutdown(ctx2)
			}()
			logger.Info().Str("endpoint", cfg.OTELEndpoint).Msg("OTel tracing aktif")
		}
	}
	// ─────────────────────────────────────────────────────────────────────

	// PostgreSQL — kasku_finance
	financePool, err := persistence.NewPostgresPool(ctx, cfg.Finance.DSN)
	if err != nil {
		logger.Fatal().Err(err).Msg("gagal koneksi ke kasku_finance")
	}
	defer financePool.Close()
	logger.Info().Msg("kasku_finance terhubung")

	// PostgreSQL — kasku_billing (untuk CreateFreeSubscription saja)
	billingPool, err := persistence.NewPostgresPool(ctx, cfg.Billing.DSN)
	if err != nil {
		logger.Fatal().Err(err).Msg("gagal koneksi ke kasku_billing")
	}
	defer billingPool.Close()
	logger.Info().Msg("kasku_billing terhubung")

	// PostgreSQL — kasku_user (owner user_profiles)
	userPool, err := persistence.NewPostgresPool(ctx, cfg.User.DSN)
	if err != nil {
		logger.Fatal().Err(err).Msg("gagal koneksi ke kasku_user")
	}
	defer userPool.Close()
	logger.Info().Msg("kasku_user terhubung")

	// RabbitMQ Consumer
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

	// Repositories
	financeRepo := persistence.NewPostgresFinanceRepository(financePool)
	subscriptionRepo := persistence.NewPostgresSubscriptionRepository(billingPool)
	profileRepo := persistence.NewPostgresUserProfileRepository(userPool)

	// Use Cases
	provisionUC := usecase.NewProvisionTenantUseCase(financeRepo, subscriptionRepo, profileRepo, logger)

	// Event Handler adapter
	eventHandler := &eventHandlerAdapter{provisionUC: provisionUC, log: logger}

	// Mulai consume events
	consumeCtx, consumeCancel := context.WithCancel(ctx)
	defer consumeCancel()

	if err := consumer.StartConsuming(consumeCtx, eventHandler); err != nil {
		logger.Fatal().Err(err).Msg("gagal mulai consume events")
	}
	logger.Info().Msg("consumer events berjalan")

	// HTTP Server
	healthChecker := &appHealthChecker{
		financePool: financePool,
		billingPool: billingPool,
		userPool:    userPool,
		consumer:    consumer,
	}
	userHandler := handler.NewUserHandler(healthChecker, profileRepo, cfg.App.ServiceVersion, logger)
	metricsReg := obsmetrics.NewRegistry("user-service")
	metricsReg.RegisterDBPool(userPool)
	router := deliveryhttp.NewRouter(userHandler, cfg.IsDevelopment(), metricsReg, logger)

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info().Str("port", cfg.Server.Port).Msg("user-service listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("HTTP server error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	logger.Info().Msg("menerima sinyal shutdown, memulai graceful shutdown")

	consumeCancel()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("HTTP server shutdown error")
	}

	logger.Info().Msg("user-service berhenti dengan bersih")
}

func setupLogger(cfg *configs.Config) zerolog.Logger {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	level, err := zerolog.ParseLevel(cfg.App.LogLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)
	return zerolog.New(os.Stdout).With().Timestamp().Str("service", "user-service").Logger()
}

// eventHandlerAdapter mengadaptasi use case ke interface messaging.EventHandler.
type eventHandlerAdapter struct {
	provisionUC *usecase.ProvisionTenantUseCase
	log         zerolog.Logger
}

func (a *eventHandlerAdapter) HandleUserRegistered(ctx context.Context, event messaging.UserRegisteredEvent) error {
	return a.provisionUC.Execute(ctx, event.UserID, event.Email, event.Username)
}

// appHealthChecker mengimplementasikan handler.HealthChecker.
type appHealthChecker struct {
	financePool *pgxpool.Pool
	billingPool *pgxpool.Pool
	userPool    *pgxpool.Pool
	consumer    *messaging.RabbitMQConsumer
}

func (h *appHealthChecker) PingFinanceDB(ctx context.Context) error {
	return persistence.PingPostgres(ctx, h.financePool)
}

func (h *appHealthChecker) PingBillingDB(ctx context.Context) error {
	return persistence.PingPostgres(ctx, h.billingPool)
}

func (h *appHealthChecker) PingUserDB(ctx context.Context) error {
	return persistence.PingPostgres(ctx, h.userPool)
}

func (h *appHealthChecker) PingRabbitMQ() error {
	return h.consumer.Ping()
}
