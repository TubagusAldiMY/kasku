package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/TubagusAldiMY/kasku/api-gateway/configs"
	deliveryhttp "github.com/TubagusAldiMY/kasku/api-gateway/internal/delivery/http"
	"github.com/TubagusAldiMY/kasku/api-gateway/internal/delivery/http/handler"
	"github.com/TubagusAldiMY/kasku/api-gateway/internal/delivery/http/middleware"
	grpcinfra "github.com/TubagusAldiMY/kasku/api-gateway/internal/infrastructure/grpc"
	redisinfra "github.com/TubagusAldiMY/kasku/api-gateway/internal/infrastructure/redis"
	"github.com/TubagusAldiMY/kasku/api-gateway/internal/usecase"
	obsmetrics "github.com/TubagusAldiMY/kasku/observability-go/metrics"
)

func main() {
	// ── Config ──────────────────────────────────────────────────────────────
	cfg, err := configs.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("gagal load konfigurasi")
	}

	// ── Logger ──────────────────────────────────────────────────────────────
	logger := setupLogger(cfg)
	logger.Info().
		Str("service", "api-gateway").
		Str("version", cfg.App.ServiceVersion).
		Str("env", cfg.App.Env).
		Msg("api-gateway starting")

	// ── OTel Tracing ──────────────────────────────────────────────────────────
	if cfg.App.OTELEndpoint == "" {
		// Noop provider — otelgin/otelgrpc tetap bisa dipanggil tanpa panic
		otel.SetTracerProvider(noop.NewTracerProvider())
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{}, propagation.Baggage{},
		))
	} else {
		otelCtx := context.Background()
		exp, otelErr := otlptracegrpc.New(otelCtx,
			otlptracegrpc.WithEndpoint(cfg.App.OTELEndpoint),
			otlptracegrpc.WithInsecure(),
		)
		if otelErr != nil {
			logger.Warn().Err(otelErr).Msg("OTel exporter init gagal, tracing dinonaktifkan")
			otel.SetTracerProvider(noop.NewTracerProvider())
			otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
				propagation.TraceContext{}, propagation.Baggage{},
			))
		} else {
			res, _ := resource.New(otelCtx, resource.WithAttributes(
				semconv.ServiceName("api-gateway"),
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
				propagation.TraceContext{}, propagation.Baggage{},
			))
			defer func() {
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if shutdownErr := tp.Shutdown(shutdownCtx); shutdownErr != nil {
					logger.Warn().Err(shutdownErr).Msg("OTel tracer shutdown error")
				}
			}()
			logger.Info().Str("endpoint", cfg.App.OTELEndpoint).Msg("OTel tracing aktif")
		}
	}

	// ── Redis ─────────────────────────────────────────────────────────────────
	ctx := context.Background()
	redisClient := redisinfra.NewRedisClient(cfg.Redis.Addr, cfg.Redis.Password)
	if err := redisinfra.Ping(ctx, redisClient); err != nil {
		logger.Fatal().Err(err).Msg("gagal koneksi ke Redis")
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			logger.Warn().Err(err).Msg("gagal menutup Redis client")
		}
	}()
	logger.Info().Msg("Redis terhubung")

	// ── Billing gRPC Client ────────────────────────────────────────────────────
	billingClient, err := grpcinfra.NewBillingClient(cfg.Billing.GRPCAddr, cfg.Billing.Timeout)
	if err != nil {
		// Billing gRPC tidak mandatory saat startup — gateway bisa jalan dengan FREE tier fallback
		logger.Warn().Err(err).Msg("gagal inisialisasi billing gRPC client, fallback ke FREE tier")
		billingClient = nil
	} else {
		defer func() {
			if err := billingClient.Close(); err != nil {
				logger.Error().Err(err).Msg("gagal tutup koneksi gRPC billing")
			}
		}()
		logger.Info().Str("addr", cfg.Billing.GRPCAddr).Msg("billing gRPC client siap")
	}

	// ── Use Cases ─────────────────────────────────────────────────────────────
	tokenBlacklist := redisinfra.NewTokenBlacklist(redisClient)
	rateLimiter := redisinfra.NewRateLimiter(redisClient)

	jwtVerifyUC := usecase.NewJWTVerificationUseCase(cfg.JWT.PublicKey, tokenBlacklist)
	rateLimitUC := usecase.NewRateLimitUseCase(rateLimiter)

	// ── Tier Limits Provider (billing gRPC dengan fallback FREE) ───────────────
	var tierProvider middleware.TierLimitsProvider
	if billingClient != nil {
		tierProvider = billingClient
	} else {
		tierProvider = &freeTierProvider{}
	}

	// ── Handlers ──────────────────────────────────────────────────────────────
	healthHandler := handler.NewHealthHandler(cfg.App.ServiceVersion)

	upstreams := map[string]string{
		"auth":         cfg.Proxy.AuthServiceURL,
		"user":         cfg.Proxy.UserServiceURL,
		"billing":      cfg.Proxy.BillingServiceURL,
		"finance":      cfg.Proxy.FinanceServiceURL,
		"transaction":  cfg.Proxy.TransactionServiceURL,
		"investment":   cfg.Proxy.InvestmentServiceURL,
		"price":        cfg.Proxy.PriceServiceURL,
		"sync":         cfg.Proxy.SyncServiceURL,
		"notification": cfg.Proxy.NotificationServiceURL,
		"admin":        cfg.Proxy.AdminServiceURL,
	}
	proxyHandler, err := handler.NewProxyHandler(upstreams, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("gagal inisialisasi proxy handler")
	}

	// ── Middleware ────────────────────────────────────────────────────────────
	authMiddleware := middleware.Auth(jwtVerifyUC, tierProvider)
	rateLimitMiddleware := middleware.RateLimit(rateLimitUC)
	corsMiddleware := middleware.CORS(cfg.CORS.AllowedOrigins)

	// ── Router ────────────────────────────────────────────────────────────────
	metricsReg := obsmetrics.NewRegistry("api-gateway")
	router := deliveryhttp.NewRouter(deliveryhttp.RouterConfig{
		HealthHandler:       healthHandler,
		ProxyHandler:        proxyHandler,
		AuthMiddleware:      authMiddleware,
		RateLimitMiddleware: rateLimitMiddleware,
		CORSMiddleware:      corsMiddleware,
		IsDev:               cfg.IsDevelopment(),
		Logger:              logger,
		Metrics:             metricsReg,
	})

	// ── HTTP Server ───────────────────────────────────────────────────────────
	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info().Str("port", cfg.Server.Port).Msg("api-gateway listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("HTTP server error")
		}
	}()

	// ── Graceful Shutdown ──────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	logger.Info().Msg("menerima sinyal shutdown, memulai graceful shutdown")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("HTTP server shutdown error")
	}

	logger.Info().Msg("api-gateway berhenti dengan bersih")
}

// setupLogger mengkonfigurasi zerolog JSON logger.
func setupLogger(cfg *configs.Config) zerolog.Logger {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	level, err := zerolog.ParseLevel(cfg.App.LogLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)
	return zerolog.New(os.Stdout).With().Timestamp().Str("service", "api-gateway").Logger()
}

// freeTierProvider adalah fallback TierLimitsProvider yang selalu mengembalikan FREE tier limits.
// Digunakan ketika billing-service tidak tersedia saat startup.
type freeTierProvider struct{}

func (p *freeTierProvider) GetTierLimits(_ context.Context, _ string) grpcinfra.TierLimits {
	return grpcinfra.TierLimits{
		MaxTransactionsPerMonth:   50,
		MaxFinancialAccounts:      3,
		MaxInvestmentInstruments:  0,
		HistoryRetentionMonths:    3,
		EmailNotificationsEnabled: false,
		ExportCsvEnabled:          false,
	}
}
