// Package grpc menyediakan gRPC server internal untuk auth-service (port :9081).
// Konvensi pattern: manual wire-encoding dengan google.golang.org/protobuf/encoding/protowire
// dan grpc.ServiceDesc literal — konsisten dengan billing-service di repo ini.
// Lihat proto/auth/v1/auth.proto sebagai source-of-truth kontrak.
package grpc

import (
	"context"
	"fmt"
	"net"

	"github.com/TubagusAldiMY/kasku/auth-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/auth-service/internal/usecase"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// ─── Provider interfaces ──────────────────────────────────────────────────────
// Interface didefinisikan di consumer (server) sesuai prinsip Go (accept interfaces).
// Implementasi konkret hidup di usecase / repository / redis packages.

// TokenValidator memvalidasi access token (signature + expiry + blacklist).
// Diimplementasikan oleh usecase.ValidateAccessTokenUseCase.
type TokenValidator interface {
	Execute(ctx context.Context, tokenString string) (*usecase.JWTClaims, error)
}

// UserLookup mengambil entitas User berdasarkan ID atau email.
// Diimplementasikan oleh repository.UserRepository (subset method-nya).
type UserLookup interface {
	FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
}

// TokenRevoker mencabut SEMUA refresh token aktif user.
// Diimplementasikan oleh repository.RefreshTokenRepository.RevokeAllActiveByUserID.
type TokenRevoker interface {
	RevokeAllActiveByUserID(ctx context.Context, userID uuid.UUID) error
}

// BlacklistChecker memeriksa JTI di Redis blacklist.
// Diimplementasikan oleh redis.TokenBlacklist.
type BlacklistChecker interface {
	IsJTIBlacklisted(ctx context.Context, jti string) (bool, error)
}

// AuthGRPCServer membungkus grpc.Server beserta dependency-nya.
type AuthGRPCServer struct {
	tokenValidator   TokenValidator
	userLookup       UserLookup
	tokenRevoker     TokenRevoker
	blacklistChecker BlacklistChecker

	log            zerolog.Logger
	internalSecret string
	// enableReflection memunculkan grpc.reflection — hanya untuk dev/staging.
	// Di production WAJIB false agar API schema tidak ter-dump ke attacker via
	// grpcurl/evans pada port internal.
	enableReflection bool

	server   *grpc.Server
	listener net.Listener
}

// NewAuthGRPCServer membuat instance AuthGRPCServer baru.
//
//	internalSecret   — shared secret untuk RPC sensitif (RevokeUserTokens).
//	                   Set "" untuk dev (akan log warning).
//	enableReflection — true hanya untuk dev. Production HARUS false.
func NewAuthGRPCServer(
	tokenValidator TokenValidator,
	userLookup UserLookup,
	tokenRevoker TokenRevoker,
	blacklistChecker BlacklistChecker,
	internalSecret string,
	enableReflection bool,
	log zerolog.Logger,
) *AuthGRPCServer {
	return &AuthGRPCServer{
		tokenValidator:   tokenValidator,
		userLookup:       userLookup,
		tokenRevoker:     tokenRevoker,
		blacklistChecker: blacklistChecker,
		internalSecret:   internalSecret,
		enableReflection: enableReflection,
		log:              log,
	}
}

// Start memulai gRPC server pada port yang diberikan, di goroutine terpisah.
// Tidak blocking — caller perlu manage shutdown via Stop().
func (s *AuthGRPCServer) Start(port string) error {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("gagal listen pada gRPC port %s: %w", port, err)
	}
	s.listener = lis

	// Interceptors disusun outer-to-inner. Recovery harus paling luar agar
	// menangkap panic di interceptor lain.
	sensitiveMethods := map[string]bool{
		"/auth.v1.AuthInternal/RevokeUserTokens": true,
	}

	s.server = grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			recoveryInterceptor(s.log),
			correlationIDInterceptor(),
			loggingInterceptor(s.log),
			internalSecretInterceptor(s.internalSecret, sensitiveMethods, s.log),
		),
	)

	s.server.RegisterService(&authInternalServiceDesc, s)

	// gRPC health check protocol — dipakai untuk readiness/liveness probe.
	healthSrv := health.NewServer()
	healthSrv.SetServingStatus("auth.v1.AuthInternal", grpc_health_v1.HealthCheckResponse_SERVING)
	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	grpc_health_v1.RegisterHealthServer(s.server, healthSrv)

	// Reflection memudahkan debugging via grpcurl/evans — HANYA di dev.
	// Di prod reflection harus disabled supaya API schema tidak ter-leak.
	if s.enableReflection {
		reflection.Register(s.server)
		s.log.Warn().Msg("gRPC reflection ENABLED — untuk dev/staging saja")
	}

	s.log.Info().Str("port", port).Msg("auth gRPC server listening")

	go func() {
		if err := s.server.Serve(lis); err != nil {
			s.log.Error().Err(err).Msg("auth gRPC server berhenti dengan error")
		}
	}()

	return nil
}

// Stop melakukan graceful shutdown — menunggu RPC in-flight selesai.
func (s *AuthGRPCServer) Stop() {
	if s.server != nil {
		s.server.GracefulStop()
	}
}
