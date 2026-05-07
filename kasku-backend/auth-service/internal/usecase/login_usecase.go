package usecase

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/TubagusAldiMY/kasku/auth-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/auth-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/auth-service/internal/domain/repository"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"
)

const (
	subscriptionTierFree = "FREE"
	tokenTypeBearear     = "Bearer"
)

// JWTClaims merupakan custom claims untuk JWT KasKu.
// Memuat informasi tenant schema dan subscription tier untuk downstream services.
type JWTClaims struct {
	jwt.RegisteredClaims
	Email            string `json:"email"`
	TenantSchema     string `json:"tenant_schema"`
	SubscriptionTier string `json:"subscription_tier"`
}

// LoginInput merupakan kredensial yang dikirimkan saat login.
type LoginInput struct {
	Email     string
	Password  string
	UserAgent string
	IPAddress string
	IsDev     bool // mode dev → cookie Secure=false
}

// LoginOutput merupakan data yang dikembalikan setelah login berhasil.
type LoginOutput struct {
	AccessToken        string
	TokenType          string
	ExpiresIn          int64 // detik
	RefreshTokenCookie RefreshTokenCookieParams
}

// RefreshTokenCookieParams berisi parameter untuk Set-Cookie header.
type RefreshTokenCookieParams struct {
	RawToken string
	MaxAge   int
	IsSecure bool
}

// LoginUseCase mengimplementasikan alur autentikasi user.
type LoginUseCase struct {
	userRepo         repository.UserRepository
	refreshTokenRepo repository.RefreshTokenRepository
	jwtPrivateKey    *rsa.PrivateKey
	accessTokenTTL   time.Duration
	refreshTokenTTL  time.Duration
	argon2Config     Argon2Config
	bruteForceMax    int16
	lockoutDuration  time.Duration
}

// NewLoginUseCase membuat instance LoginUseCase.
func NewLoginUseCase(
	userRepo repository.UserRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
	jwtPrivateKey *rsa.PrivateKey,
	accessTokenTTL time.Duration,
	refreshTokenTTL time.Duration,
	argon2Cfg Argon2Config,
	bruteForceMax int16,
	lockoutDuration time.Duration,
) *LoginUseCase {
	return &LoginUseCase{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		jwtPrivateKey:    jwtPrivateKey,
		accessTokenTTL:   accessTokenTTL,
		refreshTokenTTL:  refreshTokenTTL,
		argon2Config:     argon2Cfg,
		bruteForceMax:    bruteForceMax,
		lockoutDuration:  lockoutDuration,
	}
}

// Execute menjalankan alur login dengan proteksi brute force:
// 1. Lookup user by email
// 2. Jika tidak ditemukan: dummy verify (timing attack prevention) → 401
// 3. Cek account lock
// 4. Verify Argon2id password
// 5. Cek account active
// 6. Generate JWT access token + refresh token
// 7. Set cookie
func (uc *LoginUseCase) Execute(ctx context.Context, input LoginInput) (*LoginOutput, error) {
	user, err := uc.userRepo.FindByEmail(ctx, input.Email)
	if err != nil {
		return nil, fmt.Errorf("gagal lookup user: %w", err)
	}

	if user == nil {
		// User tidak ditemukan — jalankan dummy hash untuk mencegah timing attack
		runDummyArgon2Verify(input.Password, uc.argon2Config)
		return nil, domainerrors.ErrInvalidCredentials
	}

	// Cek account lock sebelum verify password
	now := time.Now().UTC()
	if user.IsAccountLocked(now) {
		return nil, domainerrors.ErrAccountLocked
	}

	passwordValid := verifyArgon2idPassword(input.Password, user.PasswordHash)

	if !passwordValid {
		// Password salah → increment counter + lock jika perlu
		lockoutStr := uc.lockoutDuration.String()
		if err := uc.userRepo.IncrementFailedLoginAndLockIfNeeded(ctx, user.ID, uc.bruteForceMax, lockoutStr); err != nil {
			// Log error tapi tetap return INVALID_CREDENTIALS (jangan expose internal error)
			return nil, domainerrors.ErrInvalidCredentials
		}
		return nil, domainerrors.ErrInvalidCredentials
	}

	// Password benar — cek apakah akun sudah diverifikasi
	if !user.IsActive {
		// Reset counter meski akun belum aktif (password valid)
		_ = uc.userRepo.UpdateLoginSuccess(ctx, user.ID)
		return nil, domainerrors.ErrAccountNotVerified
	}

	// Login berhasil — reset counter dan update last_login_at
	if err := uc.userRepo.UpdateLoginSuccess(ctx, user.ID); err != nil {
		return nil, fmt.Errorf("gagal update login success: %w", err)
	}

	// Generate JWT access token
	accessToken, err := uc.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("gagal generate access token: %w", err)
	}

	// Generate refresh token (32 random bytes → rawHex)
	rawToken, tokenHash, err := generateSecureTokenWithHash()
	if err != nil {
		return nil, fmt.Errorf("gagal generate refresh token: %w", err)
	}

	// Simpan refresh token ke DB
	userAgent := &input.UserAgent
	ipAddress := &input.IPAddress
	if input.UserAgent == "" {
		userAgent = nil
	}
	if input.IPAddress == "" {
		ipAddress = nil
	}

	refreshToken := &entity.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: tokenHash,
		UserAgent: userAgent,
		IPAddress: ipAddress,
		ExpiresAt: now.Add(uc.refreshTokenTTL),
		CreatedAt: now,
	}

	if err := uc.refreshTokenRepo.Create(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("gagal simpan refresh token: %w", err)
	}

	maxAge := int(uc.refreshTokenTTL.Seconds())

	return &LoginOutput{
		AccessToken: accessToken,
		TokenType:   tokenTypeBearear,
		ExpiresIn:   int64(uc.accessTokenTTL.Seconds()),
		RefreshTokenCookie: RefreshTokenCookieParams{
			RawToken: rawToken,
			MaxAge:   maxAge,
			IsSecure: !input.IsDev,
		},
	}, nil
}

// generateAccessToken membuat JWT RS256 dengan custom claims KasKu.
func (uc *LoginUseCase) generateAccessToken(user *entity.User) (string, error) {
	now := time.Now().UTC()
	jti := uuid.New().String()

	// tenant_schema = "tenant_" + user_id dengan dash diganti underscore
	tenantSchema := "tenant_" + strings.ReplaceAll(user.ID.String(), "-", "_")

	claims := JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(uc.accessTokenTTL)),
		},
		Email:            user.Email,
		TenantSchema:     tenantSchema,
		SubscriptionTier: subscriptionTierFree,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(uc.jwtPrivateKey)
	if err != nil {
		return "", fmt.Errorf("gagal sign JWT: %w", err)
	}

	return signed, nil
}

// verifyArgon2idPassword memverifikasi password terhadap hash PHC string yang tersimpan.
// Mendukung format: $argon2id$v=19$m=...,t=...,p=...$<salt>$<hash>
func verifyArgon2idPassword(password, encodedHash string) bool {
	parts := strings.Split(encodedHash, "$")
	// Format: ["", "argon2id", "v=19", "m=...,t=...,p=...", "<salt>", "<hash>"]
	if len(parts) != 6 {
		return false
	}

	if parts[1] != "argon2id" {
		return false
	}

	var memKB, timeCost uint32
	var threads uint8
	// Parse parameter: m=65536,t=3,p=4
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memKB, &timeCost, &threads); err != nil {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}

	storedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}

	keyLen := uint32(len(storedHash))
	computedHash := argon2.IDKey([]byte(password), salt, timeCost, memKB, threads, keyLen)

	// Constant-time comparison untuk mencegah timing attack
	return constantTimeEqual(computedHash, storedHash)
}

// constantTimeEqual membandingkan dua slice bytes dalam waktu konstan.
func constantTimeEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := range a {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}

// runDummyArgon2Verify menjalankan operasi hash Argon2id palsu untuk memastikan
// response time tidak berbeda antara "user tidak ditemukan" dan "password salah".
// Ini mencegah timing attack berbasis waktu respons.
func runDummyArgon2Verify(password string, cfg Argon2Config) {
	dummySalt := make([]byte, 16)
	argon2.IDKey([]byte(password), dummySalt, cfg.Time, cfg.MemoryKB, cfg.Threads, cfg.KeyLength)
}
