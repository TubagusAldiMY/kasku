package grpc

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
)

// getUserTierLimitsHandler adalah unary handler untuk RPC GetUserTierLimits.
// Signature harus sesuai persis dengan grpc.MethodDesc.Handler type.
func getUserTierLimitsHandler(srv any, ctx context.Context, dec func(any) error, _ grpc.UnaryServerInterceptor) (any, error) {
	reqMsg := &rawBytesMsg{}
	if err := dec(reqMsg); err != nil {
		return nil, fmt.Errorf("gagal decode request: %w", err)
	}

	s, ok := srv.(*BillingGRPCServer)
	if !ok {
		return nil, fmt.Errorf("internal error: server type assertion gagal")
	}

	userID, err := decodeGetUserTierLimitsRequest(reqMsg.data)
	if err != nil {
		return nil, fmt.Errorf("gagal decode proto request: %w", err)
	}

	limits, err := s.tierLimitsUC.Execute(ctx, userID)
	if err != nil {
		s.log.Error().Err(err).Str("user_id", userID).Msg("gagal mengambil tier limits")
		return nil, toStatus(err)
	}

	respData := encodeTierLimitsResponse(limits)
	return &rawBytesMsg{data: respData}, nil
}
