package entity

import (
	"time"

	"github.com/google/uuid"
)

// User merepresentasikan entitas pengguna dalam domain auth.
// Entitas ini murni domain — tidak bergantung pada framework apapun.
type User struct {
	ID               uuid.UUID
	Email            string
	Username         string
	PasswordHash     string
	IsActive         bool
	EmailVerified    bool
	FailedLoginCount int16
	LockedUntil      *time.Time
	LastLoginAt      *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// IsAccountLocked memeriksa apakah akun sedang dalam status terkunci.
func (u *User) IsAccountLocked(now time.Time) bool {
	return u.LockedUntil != nil && u.LockedUntil.After(now)
}

// RefreshToken merepresentasikan refresh token yang tersimpan di database.
// Hanya token_hash yang disimpan — raw token tidak pernah disimpan.
type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	UserAgent *string
	IPAddress *string
	ExpiresAt time.Time
	IsRevoked bool
	RevokedAt *time.Time
	CreatedAt time.Time
}

// IsExpired memeriksa apakah refresh token sudah kadaluwarsa.
func (rt *RefreshToken) IsExpired(now time.Time) bool {
	return rt.ExpiresAt.Before(now)
}

// EmailVerification merepresentasikan token verifikasi email.
// Hanya token_hash yang disimpan di database.
type EmailVerification struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	TokenHash  string
	ExpiresAt  time.Time
	VerifiedAt *time.Time
	CreatedAt  time.Time
}

// IsExpired memeriksa apakah token verifikasi sudah kadaluwarsa.
func (ev *EmailVerification) IsExpired(now time.Time) bool {
	return ev.ExpiresAt.Before(now)
}

// IsUsed memeriksa apakah token verifikasi sudah digunakan.
func (ev *EmailVerification) IsUsed() bool {
	return ev.VerifiedAt != nil
}

// PasswordResetToken merepresentasikan token reset password.
// Hanya token_hash yang disimpan di database.
type PasswordResetToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

// IsExpired memeriksa apakah token reset sudah kadaluwarsa.
func (pr *PasswordResetToken) IsExpired(now time.Time) bool {
	return pr.ExpiresAt.Before(now)
}

// IsUsed memeriksa apakah token reset sudah digunakan.
func (pr *PasswordResetToken) IsUsed() bool {
	return pr.UsedAt != nil
}
