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
	"github.com/TubagusAldiMY/kasku/billing-service/internal/infrastructure/cleanup"
	billinggrpc "github.com/TubagusAldiMY/kasku/billing-service/internal/infrastructure/grpc"
	"github.com/TubagusAldiMY/kasku/billing-service/internal/infrastructure/messaging"
	"github.com/TubagusAldiMY/kasku/billing-service/internal/infrastructure/outbox"
	paymentinfra "github.com/TubagusAldiMY/kasku/billing-service/internal/infrastructure/payment"
	"github.com/TubagusAldiMY/kasku/billing-service/internal/infrastructure/persistence"
	"github.com/TubagusAldiMY/kasku/billing-service/internal/usecase"
	obsmetrics "github.com/TubagusAldiMY/kasku/observability-go/metrics"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
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
		startupLog := zerolog.New(os.Stdout).With().Timestamp().Logger()
		startupLog.Fatal().Err(err).Msg("gagal load konfigurasi")
	}

	logger := buildLogger(cfg)
	logger.Info().
		Str("service", "billing-service").
		Str("version", cfg.App.ServiceVersion).
		Str("env", cfg.App.Env).
		Msg("billing-service starting")

	// Inisialisasi OpenTelemetry tracing.
	// Jika OTEL_EXPORTER_OTLP_ENDPOINT tidak diset, gunakan noop provider agar tidak ada overhead.
	if cfg.App.OTELEndpoint == "" {
		otel.SetTracerProvider(noop.NewTracerProvider())
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		))
		logger.Info().Msg("OTel tracing nonaktif (OTEL_EXPORTER_OTLP_ENDPOINT tidak diset)")
	} else {
		otelCtx := context.Background()
		exp, otelErr := otlptracehttp.New(
			otelCtx,
			otlptracehttp.WithEndpoint(cfg.App.OTELEndpoint),
			otlptracehttp.WithInsecure(),
		)
		if otelErr != nil {
			logger.Warn().Err(otelErr).Msg("OTel exporter init gagal, tracing dinonaktifkan")
			otel.SetTracerProvider(noop.NewTracerProvider())
			otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
				propagation.TraceContext{},
				propagation.Baggage{},
			))
		} else {
			res, _ := resource.New(otelCtx,
				resource.WithAttributes(
					semconv.ServiceName("billing-service"),
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
				shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer shutCancel()
				_ = tp.Shutdown(shutCtx)
			}()
			logger.Info().Str("endpoint", cfg.App.OTELEndpoint).Msg("OTel tracing aktif")
		}
	}

	// Jalankan migration sebelum menerima traffic apapun
	logger.Info().Msg("menjalankan database migrations")
	if err := persistence.RunMigrations(cfg.Postgres.DSN); err != nil {
		logger.Fatal().Err(err).Msg("gagal menjalankan database migrations")
	}
	logger.Info().Msg("database migrations selesai")

	ctx := context.Background()

	// Connection pool PostgreSQL
	pool, err := persistence.NewPostgresPool(ctx, cfg.Postgres.DSN)
	if err != nil {
		logger.Fatal().Err(err).Msg("gagal membuat koneksi ke PostgreSQL")
	}
	defer pool.Close()
	logger.Info().Msg("PostgreSQL terhubung")

	// RabbitMQ publisher untuk publish event subscription.expired/expiring lewat outbox.
	publisher, err := messaging.NewRabbitMQPublisher(cfg.RabbitMQ.URL)
	if err != nil {
		logger.Fatal().Err(err).Msg("gagal koneksi ke RabbitMQ")
	}
	defer func() { _ = publisher.Close() }()
	logger.Info().Msg("RabbitMQ terhubung")

	// Outbox dispatcher — jalan di goroutine terpisah, dihentikan via outboxCancel().
	outboxCtx, outboxCancel := context.WithCancel(context.Background())
	defer outboxCancel()
	outbox.NewDispatcher(pool, publisher, logger).Start(outboxCtx)
	logger.Info().Msg("outbox dispatcher started")

	// Cleanup job retention untuk outbox_events.
	cleanupCtx, cleanupCancel := context.WithCancel(context.Background())
	defer cleanupCancel()
	cleanupJob := cleanup.NewCleanupJob(pool, logger, cfg.Cleanup.Interval, cfg.Cleanup.DryRun)
	go cleanupJob.Run(cleanupCtx)

	// Wiring dependency injection: repository → infrastructure → use case → handler.

	// Repositories
	subRepo := persistence.NewPostgresSubscriptionRepository(pool)
	paymentRepo := persistence.NewPostgresPaymentRepository(pool)

	// Payment Orchestrator HTTP client — menggunakan API key dari config (tidak pernah hardcoded)
	orchestratorClient := paymentinfra.NewHTTPOrchestratorClient(
		cfg.Payment.OrchestratorBaseURL,
		cfg.Payment.OrchestratorAPIKey,
	)

	// Use cases
	getTierLimitsUC := usecase.NewGetTierLimitsUseCase(subRepo)
	listPlansUC := usecase.NewListPlansUseCase(subRepo)
	getSubscriptionUC := usecase.NewGetSubscriptionUseCase(subRepo)
	expireSubscriptionsUC := usecase.NewExpireSubscriptionsUseCase(subRepo, logger)
	createPaymentUC := usecase.NewCreateSubscriptionPaymentUseCase(subRepo, paymentRepo, orchestratorClient, logger)
	handleWebhookUC := usecase.NewHandlePaymentWebhookUseCase(paymentRepo, subRepo, logger)

	// Mulai gRPC server pada port 9083
	grpcServer := billinggrpc.NewBillingGRPCServer(getTierLimitsUC, logger, cfg.IsDevelopment())
	if err := grpcServer.Start(cfg.Server.GRPCPort); err != nil {
		logger.Fatal().Err(err).Msg("gagal memulai gRPC server")
	}
	defer grpcServer.Stop()
	logger.Info().Str("port", cfg.Server.GRPCPort).Msg("gRPC server berjalan")

	// HTTP server + composite health checker.
	healthChecker := newHealthChecker(pool, publisher)
	billingHandler := handler.NewBillingHandler(
		healthChecker,
		listPlansUC,
		getSubscriptionUC,
		createPaymentUC,
		handleWebhookUC,
		cfg.Payment.WebhookSecret,
		cfg.App.ServiceVersion,
		logger,
	)
	metricsReg := obsmetrics.NewRegistry("billing-service")
	metricsReg.RegisterDBPool(pool)
	router := deliveryhttp.NewRouter(billingHandler, cfg.IsDevelopment(), metricsReg, logger)

	httpServer := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info().Str("port", cfg.Server.Port).Msg("billing-service HTTP listening")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("HTTP server berhenti dengan error")
		}
	}()

	// Background job — update expired subscriptions + publish event via outbox.
	go runSubscriptionExpiryCheck(ctx, expireSubscriptionsUC, cfg.App.SubscriptionCheckInterval, logger)

	// Tunggu sinyal shutdown.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	logger.Info().Msg("menerima sinyal shutdown, memulai graceful shutdown (30s timeout)")

	outboxCancel()
	cleanupCancel()

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

// appHealthChecker menggabungkan ping pgxpool + RabbitMQ publisher untuk health endpoint.
type appHealthChecker struct {
	pool      *pgxpool.Pool
	publisher *messaging.RabbitMQPublisher
}

func newHealthChecker(pool *pgxpool.Pool, publisher *messaging.RabbitMQPublisher) *appHealthChecker {
	return &appHealthChecker{pool: pool, publisher: publisher}
}

func (h *appHealthChecker) PingPostgres(ctx context.Context) error {
	return persistence.PingPostgres(ctx, h.pool)
}

func (h *appHealthChecker) PingRabbitMQ() error {
	return h.publisher.Ping()
}

// runSubscriptionExpiryCheck adalah background job yang berjalan periodik untuk
// memanggil ExpireSubscriptionsUseCase. Usecase yang menjamin transaksi atomic
// UPDATE+outbox INSERT supaya event tidak hilang setelah crash.
func runSubscriptionExpiryCheck(
	ctx context.Context,
	uc usecase.ExpireSubscriptionsUseCase,
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
			processed, err := uc.Execute(ctx)
			if err != nil {
				log.Error().Err(err).Msg("subscription expiry check pass gagal")
				continue
			}
			if processed > 0 {
				log.Info().Int("count", processed).Msg("subscription expired berhasil diupdate")
			}
		}
	}
}
