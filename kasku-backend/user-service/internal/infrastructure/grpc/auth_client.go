// Package grpc berisi adapter gRPC client dari user-service ke service lain.
package grpc

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	domainerrors "github.com/TubagusAldiMY/kasku/user-service/internal/domain/errors"
	authv1 "github.com/TubagusAldiMY/kasku/user-service/proto/auth/v1"
)

// metadataKeyInternalSecret harus sama persis dengan yang diperiksa
// auth-service/internal/infrastructure/grpc/interceptors.go.
const metadataKeyInternalSecret = "x-internal-secret"

// callTimeout membatasi satu panggilan rename. Rename terjadi di jalur request
// HTTP yang ditunggu user, jadi lebih baik gagal cepat daripada menggantung.
const callTimeout = 5 * time.Second

// AuthClient adalah adapter tipis ke auth-service untuk kebutuhan user-service.
type AuthClient struct {
	conn   *grpc.ClientConn
	client authv1.AuthInternalClient
	secret string
}

// NewAuthClient membuka koneksi ke auth-service.
//
// Koneksi gRPC bersifat lazy: Dial tidak menyentuh jaringan, jadi kegagalan
// auth-service saat startup tidak menggagalkan boot user-service — ia baru
// terlihat saat rename benar-benar dipanggil, dan di situ errornya sudah
// dipetakan jadi pesan yang bisa dipahami.
func NewAuthClient(addr, internalSecret string) (*AuthClient, error) {
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()), // jaringan internal Docker
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),       // trace ikut menyeberang service
	)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat client auth-service di %s: %w", addr, err)
	}
	return &AuthClient{conn: conn, client: authv1.NewAuthInternalClient(conn), secret: internalSecret}, nil
}

// Close menutup koneksi; dipanggil saat graceful shutdown.
func (c *AuthClient) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

// UpdateUsername mengganti username di tabel users milik auth-service.
//
// Error gRPC diterjemahkan kembali ke domain error supaya lapisan di atasnya
// tidak perlu tahu bahwa panggilan ini kebetulan lewat jaringan:
//   - AlreadyExists  → ErrUsernameTaken (409, bisa ditindaklanjuti user)
//   - NotFound       → ErrUserNotFound
//   - InvalidArgument→ ErrUsernameInvalid
//
// Sisanya dibiarkan sebagai error terbungkus agar tercatat sebagai kegagalan
// server sungguhan, bukan kesalahan input user.
func (c *AuthClient) UpdateUsername(ctx context.Context, userID, username string) error {
	ctx, cancel := context.WithTimeout(ctx, callTimeout)
	defer cancel()

	// Secret dikirim per panggilan, bukan disimpan di koneksi, supaya nilainya
	// tidak ikut ter-log oleh interceptor yang mencetak opsi dial.
	ctx = metadata.AppendToOutgoingContext(ctx, metadataKeyInternalSecret, c.secret)

	_, err := c.client.UpdateUsername(ctx, &authv1.UpdateUsernameRequest{
		UserID:   userID,
		Username: username,
	})
	if err == nil {
		return nil
	}

	switch status.Code(err) {
	case codes.AlreadyExists:
		return domainerrors.ErrUsernameTaken
	case codes.NotFound:
		return domainerrors.ErrUserNotFound
	case codes.InvalidArgument:
		return domainerrors.ErrUsernameInvalid
	default:
		return fmt.Errorf("auth-service UpdateUsername: %w", err)
	}
}
