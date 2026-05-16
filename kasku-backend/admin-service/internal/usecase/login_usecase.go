package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/admin-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/repository"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/infrastructure/jwt"
)

// LoginInput dikirim handler login.
type LoginInput struct {
	Username string
	Password string
	IP       string
}

// LoginOutput dikembalikan setelah login sukses.
type LoginOutput struct {
	AccessToken string
	TokenType   string
	ExpiresIn   int64
	Admin       *entity.AdminUser
}

// LoginUseCase adalah kontrak alur login admin.
type LoginUseCase interface {
	Execute(ctx context.Context, in LoginInput) (*LoginOutput, error)
}

type loginUseCase struct {
	repo   repository.AdminUserRepository
	signer *jwt.Signer
	argon2 Argon2Config
	audit  *AuditLogger
}

// NewLoginUseCase membuat instance.
func NewLoginUseCase(
	repo repository.AdminUserRepository,
	signer *jwt.Signer,
	argon2Cfg Argon2Config,
	audit *AuditLogger,
) LoginUseCase {
	return &loginUseCase{repo: repo, signer: signer, argon2: argon2Cfg, audit: audit}
}

// Execute:
// 1. Lookup admin by username (case-insensitive)
// 2. Jika tidak ada → dummy verify untuk timing-safe, return INVALID_CREDENTIALS
// 3. Cek is_active
// 4. Verify password Argon2id (constant-time)
// 5. Generate HS256 JWT, update last_login_at, audit LOGIN
func (uc *loginUseCase) Execute(ctx context.Context, in LoginInput) (*LoginOutput, error) {
	admin, err := uc.repo.FindByUsername(ctx, in.Username)
	if err != nil {
		return nil, fmt.Errorf("gagal lookup admin: %w", err)
	}

	if admin == nil {
		runDummyVerify(in.Password, uc.argon2)
		return nil, domainerrors.ErrInvalidCredentials
	}

	if !admin.IsActive {
		return nil, domainerrors.ErrAdminInactive
	}

	if !VerifyPassword(in.Password, admin.PasswordHash) {
		return nil, domainerrors.ErrInvalidCredentials
	}

	now := time.Now().UTC()
	token, jti, _, err := uc.signer.Sign(admin, now)
	if err != nil {
		return nil, fmt.Errorf("gagal sign admin JWT: %w", err)
	}

	if err := uc.repo.UpdateLastLogin(ctx, admin.ID, now); err != nil {
		// non-fatal — log saja, jangan tolak login
		// (audit log entry tetap dicatat di bawah)
		_ = err
	}

	uc.audit.Log(ctx, AuditInput{
		AdminID: admin.ID,
		Action:  entity.AuditActionLogin,
		Metadata: map[string]any{
			"ip":  in.IP,
			"jti": jti,
		},
		Success: true,
	})

	return &LoginOutput{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   int64(uc.signer.TTL().Seconds()),
		Admin:       admin,
	}, nil
}
