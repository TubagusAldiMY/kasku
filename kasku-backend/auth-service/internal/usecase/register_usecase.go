package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"time"
	"unicode"

	"github.com/TubagusAldiMY/kasku/auth-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/auth-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/auth-service/internal/domain/repository"
	"github.com/TubagusAldiMY/kasku/auth-service/internal/infrastructure/messaging"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/argon2"
)

const (
	emailVerificationTokenTTL = 24 * time.Hour
	usernameMinLength         = 3
	usernameMaxLength         = 30
	passwordMinLength         = 8
)

var (
	emailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
)

// Argon2Config berisi parameter untuk hashing Argon2id.
type Argon2Config struct {
	Time      uint32
	MemoryKB  uint32
	Threads   uint8
	KeyLength uint32
}

// RegisterInput merupakan data yang dibutuhkan untuk registrasi user.
type RegisterInput struct {
	Email    string
	Username string
	Password string
}

// RegisterOutput merupakan data yang dikembalikan setelah registrasi berhasil.
type RegisterOutput struct {
	UserID   uuid.UUID
	Email    string
	Username string
}

// RegisterUseCase mengimplementasikan alur registrasi user baru.
type RegisterUseCase struct {
	pool         *pgxpool.Pool
	userRepo     repository.UserRepository
	publisher    messaging.EventPublisher
	argon2Config Argon2Config
}

// NewRegisterUseCase membuat instance RegisterUseCase dengan semua dependency.
func NewRegisterUseCase(
	pool *pgxpool.Pool,
	userRepo repository.UserRepository,
	publisher messaging.EventPublisher,
	argon2Cfg Argon2Config,
) *RegisterUseCase {
	return &RegisterUseCase{
		pool:         pool,
		userRepo:     userRepo,
		publisher:    publisher,
		argon2Config: argon2Cfg,
	}
}

// Execute menjalankan alur registrasi:
// 1. Validasi input
// 2. Cek keunikan email dan username
// 3. Hash password dengan Argon2id
// 4. DB transaction: INSERT user + INSERT email_verification
// 5. Publish event user.registered
// 6. Rollback jika publish gagal
func (uc *RegisterUseCase) Execute(ctx context.Context, input RegisterInput) (*RegisterOutput, error) {
	if err := validateRegisterInput(input); err != nil {
		return nil, err
	}

	emailExists, err := uc.userRepo.ExistsByEmail(ctx, input.Email)
	if err != nil {
		return nil, fmt.Errorf("gagal cek keberadaan email: %w", err)
	}
	if emailExists {
		return nil, domainerrors.ErrEmailAlreadyExists
	}

	usernameExists, err := uc.userRepo.ExistsByUsername(ctx, input.Username)
	if err != nil {
		return nil, fmt.Errorf("gagal cek keberadaan username: %w", err)
	}
	if usernameExists {
		return nil, domainerrors.ErrUsernameAlreadyExists
	}

	passwordHash, err := hashPasswordArgon2id(input.Password, uc.argon2Config)
	if err != nil {
		return nil, fmt.Errorf("gagal hash password: %w", err)
	}

	// Generate raw token dan hash-nya untuk verifikasi email
	rawToken, tokenHash, err := generateSecureTokenWithHash()
	if err != nil {
		return nil, fmt.Errorf("gagal generate verification token: %w", err)
	}

	now := time.Now().UTC()
	newUser := &entity.User{
		ID:               uuid.New(),
		Email:            input.Email,
		Username:         input.Username,
		PasswordHash:     passwordHash,
		IsActive:         false,
		EmailVerified:    false,
		FailedLoginCount: 0,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	emailVerification := &entity.EmailVerification{
		ID:        uuid.New(),
		UserID:    newUser.ID,
		TokenHash: tokenHash,
		ExpiresAt: now.Add(emailVerificationTokenTTL),
		CreatedAt: now,
	}

	// Buka DB transaction: INSERT user + INSERT email_verification harus atomic
	tx, err := uc.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("gagal memulai transaksi registrasi: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `
		INSERT INTO public.users
			(id, email, username, password_hash, is_active, email_verified, failed_login_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`,
		newUser.ID, newUser.Email, newUser.Username, newUser.PasswordHash,
		newUser.IsActive, newUser.EmailVerified, newUser.FailedLoginCount,
		newUser.CreatedAt, newUser.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("gagal insert user dalam transaksi: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO public.email_verifications (id, user_id, token_hash, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`,
		emailVerification.ID, emailVerification.UserID, emailVerification.TokenHash,
		emailVerification.ExpiresAt, emailVerification.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("gagal insert email verification dalam transaksi: %w", err)
	}

	// Publish event SEBELUM commit — jika publish gagal, defer rollback membatalkan INSERT
	event := messaging.UserRegisteredEvent{
		UserID:            newUser.ID.String(),
		Email:             input.Email,
		Username:          input.Username,
		VerificationToken: rawToken,
	}

	if err := uc.publisher.PublishUserRegistered(ctx, event); err != nil {
		return nil, domainerrors.ErrServiceUnavailable
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("gagal commit transaksi registrasi: %w", err)
	}

	return &RegisterOutput{
		UserID:   newUser.ID,
		Email:    newUser.Email,
		Username: newUser.Username,
	}, nil
}

// validateRegisterInput memvalidasi semua field input registrasi.
func validateRegisterInput(input RegisterInput) error {
	if !emailRegex.MatchString(input.Email) {
		return fmt.Errorf("%w: format email tidak valid", domainerrors.ErrValidation)
	}

	usernameLen := len(input.Username)
	if usernameLen < usernameMinLength || usernameLen > usernameMaxLength {
		return fmt.Errorf("%w: username harus %d-%d karakter", domainerrors.ErrValidation, usernameMinLength, usernameMaxLength)
	}
	if !usernameRegex.MatchString(input.Username) {
		return fmt.Errorf("%w: username hanya boleh berisi huruf, angka, dan underscore", domainerrors.ErrValidation)
	}

	return validatePassword(input.Password)
}

// validatePassword memvalidasi kekuatan password.
func validatePassword(password string) error {
	if len(password) < passwordMinLength {
		return domainerrors.ErrPasswordTooShort
	}

	var hasUpper, hasLower, hasDigit bool
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		}
	}

	if !hasUpper || !hasLower || !hasDigit {
		return domainerrors.ErrPasswordTooWeak
	}

	return nil
}

// hashPasswordArgon2id menghasilkan hash Argon2id dengan format PHC string:
// $argon2id$v=19$m=65536,t=3,p=4$<base64salt>$<base64hash>
func hashPasswordArgon2id(password string, cfg Argon2Config) (string, error) {
	salt := make([]byte, 16)
	if _, err := randReadFull(salt); err != nil {
		return "", fmt.Errorf("gagal generate salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, cfg.Time, cfg.MemoryKB, cfg.Threads, cfg.KeyLength)

	// Format PHC string sesuai spec
	encoded := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		cfg.MemoryKB,
		cfg.Time,
		cfg.Threads,
		base64Encode(salt),
		base64Encode(hash),
	)

	return encoded, nil
}

// generateSecureTokenWithHash menghasilkan 32 random bytes → rawHex (64 chars) dan SHA256 hash-nya.
// rawHex dikirim ke user, tokenHash disimpan di DB.
func generateSecureTokenWithHash() (rawToken, tokenHash string, err error) {
	rawBytes := make([]byte, 32)
	if _, err := randReadFull(rawBytes); err != nil {
		return "", "", fmt.Errorf("gagal generate secure token: %w", err)
	}

	rawToken = hex.EncodeToString(rawBytes) // 64 chars
	h := sha256.Sum256([]byte(rawToken))
	tokenHash = hex.EncodeToString(h[:]) // 64 chars

	return rawToken, tokenHash, nil
}

// hashToken menghasilkan SHA256 hash dari raw token string (untuk lookup).
func hashToken(rawToken string) string {
	h := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(h[:])
}
