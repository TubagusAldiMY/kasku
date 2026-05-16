package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/TubagusAldiMY/kasku/auth-service/internal/domain/entity"
	domainerrors "github.com/TubagusAldiMY/kasku/auth-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/auth-service/internal/usecase"
	"github.com/TubagusAldiMY/kasku/auth-service/tests/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestResetPasswordUseCase_Execute(t *testing.T) {
	t.Parallel()

	argon2Cfg := usecase.Argon2Config{Time: 1, MemoryKB: 64 * 1024, Threads: 4, KeyLength: 32}
	rawToken := "reset-token-xyz"
	tokenHash := usecase.HashTokenForTest(rawToken)
	tokenID := uuid.New()
	userID := uuid.New()

	tests := []struct {
		name      string
		rawToken  string
		newPass   string
		setupMock func(rr *mocks.MockPasswordResetRepository, txr *mocks.MockTransactionalResetPasswordRepository)
		wantErr   error
	}{
		{
			name:      "weak password (too short) → ErrPasswordTooShort",
			rawToken:  rawToken,
			newPass:   "Ab1",
			setupMock: func(_ *mocks.MockPasswordResetRepository, _ *mocks.MockTransactionalResetPasswordRepository) {},
			wantErr:   domainerrors.ErrPasswordTooShort,
		},
		{
			name:      "weak password (no digit) → ErrPasswordTooWeak",
			rawToken:  rawToken,
			newPass:   "NoDigitsHere",
			setupMock: func(_ *mocks.MockPasswordResetRepository, _ *mocks.MockTransactionalResetPasswordRepository) {},
			wantErr:   domainerrors.ErrPasswordTooWeak,
		},
		{
			name:     "token not found (expired/used) → ErrInvalidToken",
			rawToken: rawToken,
			newPass:  "NewPass123",
			setupMock: func(rr *mocks.MockPasswordResetRepository, _ *mocks.MockTransactionalResetPasswordRepository) {
				rr.EXPECT().FindActiveByTokenHash(gomock.Any(), tokenHash).Return(nil, nil)
			},
			wantErr: domainerrors.ErrInvalidToken,
		},
		{
			name:     "repo lookup infra error → wrapped",
			rawToken: rawToken,
			newPass:  "NewPass123",
			setupMock: func(rr *mocks.MockPasswordResetRepository, _ *mocks.MockTransactionalResetPasswordRepository) {
				rr.EXPECT().FindActiveByTokenHash(gomock.Any(), tokenHash).Return(nil, errors.New("db down"))
			},
		},
		{
			name:     "happy: tx executes → no error",
			rawToken: rawToken,
			newPass:  "ValidPass123",
			setupMock: func(rr *mocks.MockPasswordResetRepository, txr *mocks.MockTransactionalResetPasswordRepository) {
				rr.EXPECT().FindActiveByTokenHash(gomock.Any(), tokenHash).
					Return(&entity.PasswordResetToken{
						ID:        tokenID,
						UserID:    userID,
						TokenHash: tokenHash,
						ExpiresAt: time.Now().Add(1 * time.Hour),
					}, nil)
				txr.EXPECT().
					ExecuteResetPasswordTx(gomock.Any(), userID, gomock.Any(), tokenID).
					Return(nil)
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			rr := mocks.NewMockPasswordResetRepository(ctrl)
			txr := mocks.NewMockTransactionalResetPasswordRepository(ctrl)
			tt.setupMock(rr, txr)

			uc := usecase.NewResetPasswordUseCase(rr, txr, argon2Cfg)
			err := uc.Execute(context.Background(), tt.rawToken, tt.newPass)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			if tt.name == "repo lookup infra error → wrapped" {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}
