package grpc

import (
	"context"
	"errors"

	domainerrors "github.com/TubagusAldiMY/kasku/auth-service/internal/domain/errors"
	"github.com/jackc/pgx/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// toStatus memetakan error domain ke google.golang.org/grpc/status.
// Untuk error tak dikenal kembalikan codes.Internal — pesan generik agar tidak
// bocorkan internal detail; full error tetap di-log oleh interceptor.
func toStatus(err error) error {
	if err == nil {
		return nil
	}

	// Context cancellation / timeout — propagate as-is via grpc.
	if errors.Is(err, context.DeadlineExceeded) {
		return status.Error(codes.DeadlineExceeded, "deadline exceeded")
	}
	if errors.Is(err, context.Canceled) {
		return status.Error(codes.Canceled, "request canceled")
	}

	// pgx no rows — treat as NotFound (caller bisa cek pakai status.Code()).
	if errors.Is(err, pgx.ErrNoRows) {
		return status.Error(codes.NotFound, "resource tidak ditemukan")
	}

	// Domain errors → map ke kode gRPC.
	if domainErr, ok := domainerrors.IsDomainError(err); ok {
		switch domainErr {
		case domainerrors.ErrInvalidCredentials,
			domainerrors.ErrInvalidToken,
			domainerrors.ErrTokenReuseDetected:
			return status.Error(codes.Unauthenticated, domainErr.Code)
		case domainerrors.ErrAccountLocked,
			domainerrors.ErrAccountNotVerified:
			return status.Error(codes.PermissionDenied, domainErr.Code)
		case domainerrors.ErrEmailAlreadyExists,
			domainerrors.ErrUsernameAlreadyExists,
			domainerrors.ErrEmailAlreadyVerified:
			return status.Error(codes.AlreadyExists, domainErr.Code)
		case domainerrors.ErrUserNotFound:
			return status.Error(codes.NotFound, domainErr.Code)
		case domainerrors.ErrPasswordTooShort,
			domainerrors.ErrPasswordTooWeak,
			domainerrors.ErrValidation:
			return status.Error(codes.InvalidArgument, domainErr.Code)
		case domainerrors.ErrServiceUnavailable:
			return status.Error(codes.Unavailable, domainErr.Code)
		default:
			return status.Error(codes.Internal, "INTERNAL_ERROR")
		}
	}

	return status.Error(codes.Internal, "internal error")
}
