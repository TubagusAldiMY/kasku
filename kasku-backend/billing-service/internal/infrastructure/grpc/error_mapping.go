package grpc

import (
	"context"
	"errors"

	domainerrors "github.com/TubagusAldiMY/kasku/billing-service/internal/domain/errors"
	"github.com/jackc/pgx/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// toStatus memetakan error dari layer domain/usecase ke gRPC status code yang sesuai.
// Centralized supaya handler tidak duplikasi switch error.
//
//   - context.DeadlineExceeded / Canceled → DeadlineExceeded / Canceled
//   - pgx.ErrNoRows                       → NotFound
//   - ErrSubscriptionNotFound / ErrPlanNotFound → NotFound
//   - selain itu                          → Internal (detail di-log oleh caller)
func toStatus(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, "deadline exceeded")
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, "request canceled")
	case errors.Is(err, pgx.ErrNoRows):
		return status.Error(codes.NotFound, "resource tidak ditemukan")
	case errors.Is(err, domainerrors.ErrSubscriptionNotFound):
		return status.Error(codes.NotFound, domainerrors.ErrSubscriptionNotFound.Code)
	case errors.Is(err, domainerrors.ErrPlanNotFound):
		return status.Error(codes.NotFound, domainerrors.ErrPlanNotFound.Code)
	}

	return status.Error(codes.Internal, "internal error")
}
