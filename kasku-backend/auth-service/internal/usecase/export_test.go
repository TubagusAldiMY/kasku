package usecase

// File ini hanya ada di test build (suffix _test.go) — tidak pollute API publik.
// Tujuan: expose helper internal yang dibutuhkan oleh black-box test
// (package usecase_test), tanpa mengubah surface API production.

// HashTokenForTest mengekspos hashToken untuk pengetahuan token hash di test
// (mis. setup expected argument di gomock).
func HashTokenForTest(rawToken string) string {
	return hashToken(rawToken)
}

// HashPasswordForTest mengekspos hashPasswordArgon2id untuk test login flow
// yang butuh hash valid (mis. seed user dengan password tertentu).
func HashPasswordForTest(password string, cfg Argon2Config) (string, error) {
	return hashPasswordArgon2id(password, cfg)
}

// GenerateSecureTokenForTest mengekspos generateSecureTokenWithHash untuk test
// yang butuh pasangan raw+hash token deterministik.
func GenerateSecureTokenForTest() (rawToken, tokenHash string, err error) {
	return generateSecureTokenWithHash()
}

// ValidatePasswordForTest mengekspos validatePassword untuk test complexity rules.
func ValidatePasswordForTest(password string) error {
	return validatePassword(password)
}
