package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/TubagusAldiMY/kasku/admin-service/internal/domain/entity"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// AdminClaims adalah custom claims untuk admin JWT (HS256).
type AdminClaims struct {
	jwt.RegisteredClaims
	Role string `json:"role"`
}

// Signer membuat dan memverifikasi admin JWT HS256.
// Secret di-share antar instance admin-service via env var ADMIN_JWT_SECRET.
type Signer struct {
	secret []byte
	ttl    time.Duration
}

// NewSigner membuat instance Signer. Panik bila secret kosong (config error).
func NewSigner(secret string, ttl time.Duration) *Signer {
	if secret == "" {
		panic("admin JWT secret tidak boleh kosong")
	}
	return &Signer{secret: []byte(secret), ttl: ttl}
}

// Sign menerbitkan token HS256 untuk admin dengan jti random + ttl.
func (s *Signer) Sign(admin *entity.AdminUser, now time.Time) (token string, jti string, expiresAt time.Time, err error) {
	jti = uuid.New().String()
	expiresAt = now.Add(s.ttl)

	claims := AdminClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   admin.ID.String(),
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
		Role: string(admin.Role),
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := t.SignedString(s.secret)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("gagal sign admin JWT: %w", err)
	}
	return signed, jti, expiresAt, nil
}

// Verify memvalidasi tanda tangan + expiry token; mengembalikan claims bila valid.
func (s *Signer) Verify(tokenStr string) (*AdminClaims, error) {
	claims := &AdminClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected JWT signing method")
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("gagal parse admin JWT: %w", err)
	}
	if !token.Valid {
		return nil, errors.New("admin JWT tidak valid")
	}
	return claims, nil
}

// TTL mengembalikan TTL yang dikonfigurasi (dipakai handler untuk response).
func (s *Signer) TTL() time.Duration {
	return s.ttl
}
