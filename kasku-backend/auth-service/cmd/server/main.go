package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/TubagusAldiMY/kasku/auth-service/configs"
	deliveryhttp "github.com/TubagusAldiMY/kasku/auth-service/internal/delivery/http"
	"github.com/TubagusAldiMY/kasku/auth-service/internal/delivery/http/handler"
	"github.com/TubagusAldiMY/kasku/auth-service/internal/infrastructure/messaging"
	"github.com/TubagusAldiMY/kasku/auth-service/internal/infrastructure/persistence"
	redisinfra "github.com/TubagusAldiMY/kasku/auth-service/internal/infrastructure/redis"
	"github.com/TubagusAldiMY/kasku/auth-service/internal/usecase"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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
		Str("service", "auth-service").
		Str("version", cfg.App.ServiceVersion).
		Str("env", cfg.App.Env).
		Msg("auth-service starting")

	// ── Migrations ──────────────────────────────────────────────────────────
	logger.Info().Msg("menjalankan database migrations")
	if err := persistence.RunMigrations(cfg.Postgres.DSN); err != nil {
		logger.Fatal().Err(err).Msg("gagal menjalankan migrations")
	}
	logger.Info().Msg("migrations selesai")

	// ── PostgreSQL ──────────────────────────────────────────────────────────
	ctx := context.Background()
	pool, err := persistence.NewPostgresPool(ctx, cfg.Postgres.DSN)
	if err != nil {
		logger.Fatal().Err(err).Msg("gagal koneksi ke PostgreSQL")
	}
	defer pool.Close()
	logger.Info().Msg("PostgreSQL terhubung")

	// ── Redis ─────────────────────────────────────────────────────────────────
	redisClient := redisinfra.NewRedisClient(cfg.Redis.Addr, cfg.Redis.Password)
	if err := redisinfra.PingRedis(ctx, redisClient); err != nil {
		logger.Fatal().Err(err).Msg("gagal koneksi ke Redis")
	}
	defer redisClient.Close()
	logger.Info().Msg("Redis terhubung")

	// ── RabbitMQ ──────────────────────────────────────────────────────────────
	publisher, err := messaging.NewRabbitMQPublisher(cfg.RabbitMQ.URL)
	if err != nil {
		logger.Fatal().Err(err).Msg("gagal koneksi ke RabbitMQ")
	}
	defer func() {
		if err := publisher.Close(); err != nil {
			logger.Error().Err(err).Msg("gagal tutup koneksi RabbitMQ")
		}
	}()
	logger.Info().Msg("RabbitMQ terhubung")

	// ── RSA Keys ──────────────────────────────────────────────────────────────
	privKey, err := jwt.ParseRSAPrivateKeyFromPEM(cfg.JWT.PrivateKeyPEM)
	if err != nil {
		logger.Fatal().Err(err).Msg("gagal parse RSA private key")
	}
	pubKey, err := jwt.ParseRSAPublicKeyFromPEM(cfg.JWT.PublicKeyPEM)
	if err != nil {
		logger.Fatal().Err(err).Msg("gagal parse RSA public key")
	}

	// ── Repositories ──────────────────────────────────────────────────────────
	userRepo := persistence.NewPostgresUserRepository(pool)
	refreshTokenRepo := persistence.NewPostgresRefreshTokenRepository(pool)
	emailVerifRepo := persistence.NewPostgresEmailVerificationRepository(pool)
	resetRepo, resetTxRepo := persistence.NewPostgresPasswordResetRepository(pool)

	// ── Argon2 Config ─────────────────────────────────────────────────────────
	argon2Cfg := usecase.Argon2Config{
		Time:      cfg.Argon2.Time,
		MemoryKB:  cfg.Argon2.MemoryKB,
		Threads:   cfg.Argon2.Threads,
		KeyLength: cfg.Argon2.KeyLength,
	}

	// ── Use Cases ─────────────────────────────────────────────────────────────
	blacklist := redisinfra.NewTokenBlacklist(redisClient)

	registerUC := usecase.NewRegisterUseCase(pool, userRepo, publisher, argon2Cfg)
	verifyEmailUC := usecase.NewVerifyEmailUseCase(emailVerifRepo, userRepo)
	resendVerifUC := usecase.NewResendVerificationUseCase(userRepo, emailVerifRepo, publisher)
	loginUC := usecase.NewLoginUseCase(
		userRepo, refreshTokenRepo,
		privKey,
		cfg.JWT.AccessTokenTTL, cfg.JWT.RefreshTokenTTL,
		argon2Cfg,
		cfg.BruteForce.MaxAttempts, cfg.BruteForce.LockoutDuration,
	)
	refreshTokenUC := usecase.NewRefreshTokenUseCase(
		userRepo, refreshTokenRepo,
		privKey,
		cfg.JWT.AccessTokenTTL, cfg.JWT.RefreshTokenTTL,
	)
	logoutUC := usecase.NewLogoutUseCase(refreshTokenRepo, pubKey, blacklist)
	forgotPasswordUC := usecase.NewForgotPasswordUseCase(userRepo, resetRepo, publisher)
	resetPasswordUC := usecase.NewResetPasswordUseCase(resetRepo, resetTxRepo, argon2Cfg)

	// ── Handler & Router ──────────────────────────────────────────────────────
	healthChecker := newHealthChecker(pool, redisClient, publisher)

	authHandler := handler.NewAuthHandler(
		registerUC,
		verifyEmailUC,
		resendVerifUC,
		loginUC,
		refreshTokenUC,
		logoutUC,
		forgotPasswordUC,
		resetPasswordUC,
		healthChecker,
		cfg.App.ServiceVersion,
		cfg.IsDevelopment(),
		logger,
	)

	router := deliveryhttp.NewRouter(authHandler, cfg.IsDevelopment(), logger)

	// ── HTTP Server ───────────────────────────────────────────────────────────
	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info().Str("port", cfg.Server.Port).Msg("auth-service listening")
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

	logger.Info().Msg("auth-service berhenti dengan bersih")
}

// setupLogger mengkonfigurasi zerolog JSON logger.
func setupLogger(cfg *configs.Config) zerolog.Logger {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	level, err := zerolog.ParseLevel(cfg.App.LogLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)
	return zerolog.New(os.Stdout).With().Timestamp().Str("service", "auth-service").Logger()
}

// appHealthChecker mengimplementasikan handler.HealthChecker.
type appHealthChecker struct {
	pool        *pgxpool.Pool
	redisClient *redis.Client
	publisher   *messaging.RabbitMQPublisher
}

func newHealthChecker(pool *pgxpool.Pool, redisClient *redis.Client, publisher *messaging.RabbitMQPublisher) *appHealthChecker {
	return &appHealthChecker{pool: pool, redisClient: redisClient, publisher: publisher}
}

func (h *appHealthChecker) PingPostgres(ctx context.Context) error {
	return persistence.PingPostgres(ctx, h.pool)
}

func (h *appHealthChecker) PingRedis(ctx context.Context) error {
	return redisinfra.PingRedis(ctx, h.redisClient)
}

func (h *appHealthChecker) PingRabbitMQ() error {
	return h.publisher.Ping()
}
