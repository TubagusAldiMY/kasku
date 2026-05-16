package usecase

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2Config menyimpan parameter Argon2id.
type Argon2Config struct {
	Time      uint32
	MemoryKB  uint32
	Threads   uint8
	KeyLength uint32
}

// HashPassword menghasilkan PHC-string Argon2id untuk password baru.
// Format output: $argon2id$v=19$m=<mem>,t=<time>,p=<threads>$<salt>$<hash>
func HashPassword(password string, cfg Argon2Config) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("gagal generate salt: %w", err)
	}
	hash := argon2.IDKey([]byte(password), salt, cfg.Time, cfg.MemoryKB, cfg.Threads, cfg.KeyLength)
	return fmt.Sprintf(
		"$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		cfg.MemoryKB, cfg.Time, cfg.Threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

// VerifyPassword memverifikasi password terhadap PHC-string Argon2id yang tersimpan.
// Pakai constant-time comparison untuk mencegah timing attack.
func VerifyPassword(password, encodedHash string) bool {
	parts := strings.Split(encodedHash, "$")
	// Format: ["", "argon2id", "v=19", "m=...,t=...,p=...", "<salt>", "<hash>"]
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false
	}

	var memKB, timeCost uint32
	var threads uint8
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
	computed := argon2.IDKey([]byte(password), salt, timeCost, memKB, threads, keyLen)
	return constantTimeEqual(computed, storedHash)
}

// runDummyVerify menjalankan operasi hash palsu agar respon "user tidak ditemukan"
// dan "password salah" tidak berbeda waktunya (timing attack mitigation).
func runDummyVerify(password string, cfg Argon2Config) {
	dummySalt := make([]byte, 16)
	argon2.IDKey([]byte(password), dummySalt, cfg.Time, cfg.MemoryKB, cfg.Threads, cfg.KeyLength)
}

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
