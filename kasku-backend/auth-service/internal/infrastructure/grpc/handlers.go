package grpc

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// assertServer ekstrak *AuthGRPCServer dari srv parameter atau return error.
func assertServer(srv any) (*AuthGRPCServer, error) {
	s, ok := srv.(*AuthGRPCServer)
	if !ok {
		return nil, status.Error(codes.Internal, "type assertion gagal: bukan *AuthGRPCServer")
	}
	return s, nil
}

// decodeRawBytes mendekode pesan raw bytes dari dec callback.
func decodeRawBytes(dec func(any) error) ([]byte, error) {
	msg := &rawBytesMsg{}
	if err := dec(msg); err != nil {
		return nil, status.Errorf(codes.Internal, "gagal decode request: %v", err)
	}
	return msg.data, nil
}

// ─── ValidateToken ────────────────────────────────────────────────────────────

func validateTokenHandler(srv any, ctx context.Context, dec func(any) error, _ grpc.UnaryServerInterceptor) (any, error) {
	s, err := assertServer(srv)
	if err != nil {
		return nil, err
	}

	raw, err := decodeRawBytes(dec)
	if err != nil {
		return nil, err
	}
	req, err := decodeValidateTokenRequest(raw)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "request invalid: %v", err)
	}
	if req.AccessToken == "" {
		return nil, status.Error(codes.InvalidArgument, "access_token kosong")
	}

	claims, err := s.tokenValidator.Execute(ctx, req.AccessToken)
	if err != nil {
		return nil, toStatus(err)
	}

	resp := &validateTokenResponse{
		UserID:           claims.Subject,
		Email:            claims.Email,
		TenantSchema:     claims.TenantSchema,
		SubscriptionTier: claims.SubscriptionTier,
		JTI:              claims.ID,
	}
	if claims.ExpiresAt != nil {
		resp.ExpiresAtUnix = claims.ExpiresAt.Unix()
	}
	return &rawBytesMsg{data: encodeValidateTokenResponse(resp)}, nil
}

// ─── GetUserByID ──────────────────────────────────────────────────────────────

func getUserByIDHandler(srv any, ctx context.Context, dec func(any) error, _ grpc.UnaryServerInterceptor) (any, error) {
	s, err := assertServer(srv)
	if err != nil {
		return nil, err
	}

	raw, err := decodeRawBytes(dec)
	if err != nil {
		return nil, err
	}
	req, err := decodeGetUserByIDRequest(raw)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "request invalid: %v", err)
	}
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "user_id bukan UUID valid")
	}

	user, err := s.userLookup.FindByID(ctx, userID)
	if err != nil {
		return nil, toStatus(fmt.Errorf("FindByID: %w", err))
	}
	if user == nil {
		return nil, status.Error(codes.NotFound, "user tidak ditemukan")
	}

	return &rawBytesMsg{data: encodeUserResponse(&userResponse{
		ID:            user.ID.String(),
		Email:         user.Email,
		Username:      user.Username,
		IsActive:      user.IsActive,
		EmailVerified: user.EmailVerified,
		CreatedAtUnix: user.CreatedAt.Unix(),
	})}, nil
}

// ─── GetUserByEmail ───────────────────────────────────────────────────────────

func getUserByEmailHandler(srv any, ctx context.Context, dec func(any) error, _ grpc.UnaryServerInterceptor) (any, error) {
	s, err := assertServer(srv)
	if err != nil {
		return nil, err
	}

	raw, err := decodeRawBytes(dec)
	if err != nil {
		return nil, err
	}
	req, err := decodeGetUserByEmailRequest(raw)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "request invalid: %v", err)
	}
	if req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "email kosong")
	}

	user, err := s.userLookup.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, toStatus(fmt.Errorf("FindByEmail: %w", err))
	}
	if user == nil {
		return nil, status.Error(codes.NotFound, "user tidak ditemukan")
	}

	return &rawBytesMsg{data: encodeUserResponse(&userResponse{
		ID:            user.ID.String(),
		Email:         user.Email,
		Username:      user.Username,
		IsActive:      user.IsActive,
		EmailVerified: user.EmailVerified,
		CreatedAtUnix: user.CreatedAt.Unix(),
	})}, nil
}

// ─── RevokeUserTokens ─────────────────────────────────────────────────────────

func revokeUserTokensHandler(srv any, ctx context.Context, dec func(any) error, _ grpc.UnaryServerInterceptor) (any, error) {
	s, err := assertServer(srv)
	if err != nil {
		return nil, err
	}

	raw, err := decodeRawBytes(dec)
	if err != nil {
		return nil, err
	}
	req, err := decodeRevokeUserTokensRequest(raw)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "request invalid: %v", err)
	}
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "user_id bukan UUID valid")
	}

	if err := s.tokenRevoker.RevokeAllActiveByUserID(ctx, userID); err != nil {
		return nil, toStatus(fmt.Errorf("RevokeAllActiveByUserID: %w", err))
	}

	return &rawBytesMsg{data: encodeRevokeUserTokensResponse(&revokeUserTokensResponse{Success: true})}, nil
}

// ─── IsTokenBlacklisted ───────────────────────────────────────────────────────

func isTokenBlacklistedHandler(srv any, ctx context.Context, dec func(any) error, _ grpc.UnaryServerInterceptor) (any, error) {
	s, err := assertServer(srv)
	if err != nil {
		return nil, err
	}

	raw, err := decodeRawBytes(dec)
	if err != nil {
		return nil, err
	}
	req, err := decodeIsTokenBlacklistedRequest(raw)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "request invalid: %v", err)
	}
	if req.JTI == "" {
		return nil, status.Error(codes.InvalidArgument, "jti kosong")
	}

	blacklisted, err := s.blacklistChecker.IsJTIBlacklisted(ctx, req.JTI)
	if err != nil {
		return nil, toStatus(fmt.Errorf("IsJTIBlacklisted: %w", err))
	}

	return &rawBytesMsg{data: encodeIsTokenBlacklistedResponse(&isTokenBlacklistedResponse{Blacklisted: blacklisted})}, nil
}
