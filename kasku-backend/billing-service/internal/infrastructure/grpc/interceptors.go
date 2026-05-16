package grpc

import (
	"context"
	"runtime/debug"
	"time"

	"github.com/TubagusAldiMY/kasku/billing-service/internal/delivery/http/middleware"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const metadataKeyCorrelationID = "x-correlation-id"

// recoveryInterceptor menangkap panic di handler dan mengubahnya menjadi
// codes.Internal — mencegah crash proses. Harus dipasang OUTERMOST.
func recoveryInterceptor(log zerolog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Error().
					Str("method", info.FullMethod).
					Bytes("stack", debug.Stack()).
					Msgf("gRPC panic recovered: %v", r)
				err = status.Errorf(codes.Internal, "internal error")
			}
		}()
		return handler(ctx, req)
	}
}

// correlationIDInterceptor mengekstrak x-correlation-id dari metadata.
// Jika tidak ada, generate baru. Inject ke context sehingga handler dapat
// mem-propagate ke log untuk distributed tracing.
func correlationIDInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		corrID := ""
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if vals := md.Get(metadataKeyCorrelationID); len(vals) > 0 {
				corrID = vals[0]
			}
		}
		if corrID == "" {
			corrID = uuid.NewString()
		}
		ctx = middleware.ContextWithCorrelationID(ctx, corrID)
		return handler(ctx, req)
	}
}

// loggingInterceptor mencatat method, duration, dan kode status setiap RPC.
func loggingInterceptor(log zerolog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		dur := time.Since(start)

		code := status.Code(err)
		event := log.Info()
		if code != codes.OK {
			if code == codes.Internal || code == codes.Unavailable {
				event = log.Error()
			} else {
				event = log.Warn()
			}
		}

		event.
			Str("method", info.FullMethod).
			Str("grpc_code", code.String()).
			Dur("duration_ms", dur).
			Str("correlation_id", middleware.CorrelationIDFromContext(ctx)).
			Msg("grpc call")
		return resp, err
	}
}
