package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/TubagusAldiMY/kasku/admin-service/configs"
	deliveryhttp "github.com/TubagusAldiMY/kasku/admin-service/internal/delivery/http"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/delivery/http/handler"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/infrastructure/jwt"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/infrastructure/persistence"
	rdsinfra "github.com/TubagusAldiMY/kasku/admin-service/internal/infrastructure/redis"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/usecase"
	obsmetrics "github.com/TubagusAldiMY/kasku/observability-go/metrics"
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
		Str("service", "admin-service").
		Str("version", cfg.App.ServiceVersion).
		Str("env", cfg.App.Env).
		Msg("admin-service starting")

	ctx := context.Background()

	// ── Migrations ──────────────────────────────────────────────────────
	if err := persistence.RunMigrations(cfg.Postgres.AdminDSN); err != nil {
		logger.Fatal().Err(err).Msg("gagal menjalankan migration kasku_admin")
	}
	logger.Info().Msg("migration kasku_admin selesai")

	// ── Database pools ──────────────────────────────────────────────────
	adminPool, err := persistence.NewPostgresPool(ctx, cfg.Postgres.AdminDSN)
	if err != nil {
		logger.Fatal().Err(err).Msg("gagal koneksi kasku_admin")
	}
	defer adminPool.Close()

	authPool, err := persistence.NewPostgresPool(ctx, cfg.Postgres.AuthDSN)
	if err != nil {
		logger.Fatal().Err(err).Msg("gagal koneksi kasku_auth (admin DSN)")
	}
	defer authPool.Close()

	billingPool, err := persistence.NewPostgresPool(ctx, cfg.Postgres.BillingDSN)
	if err != nil {
		logger.Fatal().Err(err).Msg("gagal koneksi kasku_billing (admin DSN)")
	}
	defer billingPool.Close()

	// ── Redis ───────────────────────────────────────────────────────────
	redisClient := rdsinfra.NewClient(cfg.Redis.Addr, cfg.Redis.Password)
	defer func() {
		if err := redisClient.Close(); err != nil {
			logger.Warn().Err(err).Msg("gagal menutup Redis client")
		}
	}()
	if err := rdsinfra.Ping(ctx, redisClient); err != nil {
		logger.Fatal().Err(err).Msg("gagal ping Redis")
	}

	blacklist := rdsinfra.NewTokenBlacklist(redisClient)

	// ── JWT ─────────────────────────────────────────────────────────────
	signer := jwt.NewSigner(cfg.JWT.Secret, cfg.JWT.TTL)

	// ── Repositories ────────────────────────────────────────────────────
	adminUserRepo := persistence.NewPostgresAdminUserRepository(adminPool)
	auditLogRepo := persistence.NewPostgresAuditLogRepository(adminPool)
	userRepo := persistence.NewPostgresUserRepository(authPool) // implements UserReadRepository + UserWriteRepository
	paymentRepo := persistence.NewPostgresPaymentRepository(billingPool)
	subRepo := persistence.NewPostgresSubscriptionRepository(billingPool)

	// ── Bootstrap admin (idempotent) ────────────────────────────────────
	argon2Cfg := usecase.Argon2Config{
		Time:      cfg.Argon2.Time,
		MemoryKB:  cfg.Argon2.MemoryKB,
		Threads:   cfg.Argon2.Threads,
		KeyLength: cfg.Argon2.KeyLength,
	}
	if err := usecase.SeedBootstrapAdmin(ctx, adminUserRepo, usecase.BootstrapInput{
		Username: cfg.Bootstrap.Username,
		Password: cfg.Bootstrap.Password,
		Argon2:   argon2Cfg,
	}, logger); err != nil {
		logger.Fatal().Err(err).Msg("gagal seed bootstrap admin")
	}

	// ── Use cases ───────────────────────────────────────────────────────
	auditLogger := usecase.NewAuditLogger(auditLogRepo, logger)

	loginUC := usecase.NewLoginUseCase(adminUserRepo, signer, argon2Cfg, auditLogger)
	logoutUC := usecase.NewLogoutUseCase(blacklist, auditLogger)
	currentUC := usecase.NewGetCurrentAdminUseCase(adminUserRepo)

	listUsersUC := usecase.NewListUsersUseCase(userRepo, subRepo)
	getUserDetailUC := usecase.NewGetUserDetailUseCase(userRepo, subRepo)
	listPaymentsUC := usecase.NewListPaymentsUseCase(paymentRepo)
	statsUC := usecase.NewDashboardStatsUseCase(userRepo, paymentRepo)
	listAuditUC := usecase.NewListAuditLogUseCase(auditLogRepo)

	suspendUC := usecase.NewSuspendUserUseCase(userRepo, userRepo, auditLogger)
	activateUC := usecase.NewActivateUserUseCase(userRepo, userRepo, auditLogger)
	overrideUC := usecase.NewOverrideSubscriptionUseCase(subRepo, auditLogger)

	// ── Handlers ────────────────────────────────────────────────────────
	healthChecker := &handler.HealthChecker{
		AdminPool:   adminPool,
		AuthPool:    authPool,
		BillingPool: billingPool,
		RedisClient: redisClient,
	}
	healthHandler := handler.NewHealthHandler(cfg.App.ServiceVersion, healthChecker)
	authHandler := handler.NewAuthHandler(loginUC, logoutUC, currentUC)
	userHandler := handler.NewUserHandler(listUsersUC, getUserDetailUC, suspendUC, activateUC)
	subHandler := handler.NewSubscriptionHandler(overrideUC)
	paymentHandler := handler.NewPaymentHandler(listPaymentsUC)
	statsHandler := handler.NewStatsHandler(statsUC)
	auditLogHandler := handler.NewAuditLogHandler(listAuditUC)

	metricsReg := obsmetrics.NewRegistry("admin-service")
	metricsReg.RegisterDBPool(adminPool)
	router := deliveryhttp.NewRouter(deliveryhttp.RouterDeps{
		IsDev:           cfg.IsDevelopment(),
		Logger:          logger,
		Metrics:         metricsReg,
		HealthHandler:   healthHandler,
		AuthHandler:     authHandler,
		UserHandler:     userHandler,
		SubHandler:      subHandler,
		PaymentHandler:  paymentHandler,
		StatsHandler:    statsHandler,
		AuditLogHandler: auditLogHandler,
		JWTSigner:       signer,
		TokenBlacklist:  blacklist,
	})

	// ── HTTP server ─────────────────────────────────────────────────────
	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info().Str("port", cfg.Server.Port).Msg("admin-service HTTP server listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("HTTP server error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), gracefulShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("HTTP server shutdown error")
	}
	logger.Info().Msg("admin-service berhenti dengan bersih")
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
		Str("service", "admin-service").
		Logger()
}
