package usecase_test

import (
	"time"

	"github.com/TubagusAldiMY/kasku/admin-service/internal/infrastructure/jwt"
	"github.com/TubagusAldiMY/kasku/admin-service/internal/usecase"
	"github.com/rs/zerolog"
	"go.uber.org/mock/gomock"

	"github.com/TubagusAldiMY/kasku/admin-service/tests/mocks"
)

const testSecret = "test-hs256-secret-for-unit-tests-only"

// testSigner mengembalikan jwt.Signer dengan test secret dan 15 menit TTL.
func testSigner() *jwt.Signer {
	return jwt.NewSigner(testSecret, 15*time.Minute)
}

// testArgon2 mengembalikan Argon2Config dengan parameter sangat cepat untuk unit test.
// JANGAN dipakai di production — time=1, memory=1024 jauh di bawah standar OWASP.
func testArgon2() usecase.Argon2Config {
	return usecase.Argon2Config{Time: 1, MemoryKB: 1024, Threads: 1, KeyLength: 32}
}

// testAuditLogger mengembalikan AuditLogger dengan mocked repo.
// Caller bertanggung jawab untuk set expectation atau mengabaikan calls.
func testAuditLogger(ctrl *gomock.Controller) (*usecase.AuditLogger, *mocks.MockAuditLogRepository) {
	mockAudit := mocks.NewMockAuditLogRepository(ctrl)
	logger := usecase.NewAuditLogger(mockAudit, zerolog.Nop())
	return logger, mockAudit
}
