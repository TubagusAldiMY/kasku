package grpc

import (
	"context"
	"fmt"
	"net"

	"github.com/TubagusAldiMY/kasku/billing-service/internal/domain/entity"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// TierLimitsProvider mendefinisikan kontrak use case untuk mengambil tier limits.
// Digunakan sebagai abstraksi agar gRPC layer tidak bergantung pada implementasi konkrit.
type TierLimitsProvider interface {
	Execute(ctx context.Context, userID string) (*entity.PlanLimits, error)
}

// BillingGRPCServer mengimplementasikan billing.v1.BillingInternal gRPC service.
// Encoding/decoding dilakukan secara manual menggunakan protowire karena tidak ada
// file .proto yang di-generate — kompatibel penuh dengan api-gateway yang menggunakan
// rawBytesCodec.
type BillingGRPCServer struct {
	tierLimitsUC     TierLimitsProvider
	log              zerolog.Logger
	enableReflection bool

	server   *grpc.Server
	listener net.Listener
}

// NewBillingGRPCServer membuat instance BillingGRPCServer baru.
func NewBillingGRPCServer(tierLimitsUC TierLimitsProvider, log zerolog.Logger, enableReflection bool) *BillingGRPCServer {
	return &BillingGRPCServer{
		tierLimitsUC:     tierLimitsUC,
		log:              log,
		enableReflection: enableReflection,
	}
}

// Start memulai gRPC server pada port yang diberikan, di goroutine terpisah.
// Tidak blocking — caller perlu manage shutdown via Stop().
func (s *BillingGRPCServer) Start(port string) error {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("gagal listen pada gRPC port %s: %w", port, err)
	}
	s.listener = lis

	// Interceptors disusun outer-to-inner. Recovery WAJIB paling luar agar
	// menangkap panic dari interceptor & handler lainnya.
	// otelgrpc.NewServerHandler() dipasang sebagai StatsHandler agar trace context
	// dipropagasi dari caller (api-gateway) ke server tanpa mengganggu interceptor chain.
	s.server = grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			recoveryInterceptor(s.log),
			correlationIDInterceptor(),
			loggingInterceptor(s.log),
		),
	)

	s.server.RegisterService(&billingInternalServiceDesc, s)

	// gRPC health check protocol — dipakai untuk readiness/liveness probe k8s.
	healthSrv := health.NewServer()
	healthSrv.SetServingStatus("billing.v1.BillingInternal", grpc_health_v1.HealthCheckResponse_SERVING)
	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	grpc_health_v1.RegisterHealthServer(s.server, healthSrv)

	// Reflection memudahkan debugging via grpcurl/evans — HANYA di dev.
	// Di prod reflection harus disabled supaya API schema tidak ter-leak.
	if s.enableReflection {
		reflection.Register(s.server)
		s.log.Warn().Msg("gRPC reflection ENABLED — untuk dev/staging saja")
	}

	s.log.Info().Str("port", port).Msg("billing gRPC server listening")

	go func() {
		if err := s.server.Serve(lis); err != nil {
			s.log.Error().Err(err).Msg("billing gRPC server berhenti dengan error")
		}
	}()

	return nil
}

// Stop melakukan graceful shutdown — menunggu RPC in-flight selesai.
func (s *BillingGRPCServer) Stop() {
	if s.server != nil {
		s.server.GracefulStop()
	}
}
