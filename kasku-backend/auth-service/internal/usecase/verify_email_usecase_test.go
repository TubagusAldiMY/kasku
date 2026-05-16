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

func TestVerifyEmailUseCase_Execute(t *testing.T) {
	t.Parallel()

	rawToken := "abc123def456"
	tokenHash := usecase.HashTokenForTest(rawToken)
	verifID := uuid.New()
	userID := uuid.New()

	tests := []struct {
		name      string
		token     string
		setupMock func(ev *mocks.MockEmailVerificationRepository, ur *mocks.MockUserRepository)
		wantErr   error
	}{
		{
			name:      "empty token → ErrInvalidToken",
			token:     "",
			setupMock: func(_ *mocks.MockEmailVerificationRepository, _ *mocks.MockUserRepository) {},
			wantErr:   domainerrors.ErrInvalidToken,
		},
		{
			name:  "token not found (expired/used) → ErrInvalidToken",
			token: rawToken,
			setupMock: func(ev *mocks.MockEmailVerificationRepository, _ *mocks.MockUserRepository) {
				ev.EXPECT().FindActiveByTokenHash(gomock.Any(), tokenHash).Return(nil, nil)
			},
			wantErr: domainerrors.ErrInvalidToken,
		},
		{
			name:  "happy: mark verified + update user",
			token: rawToken,
			setupMock: func(ev *mocks.MockEmailVerificationRepository, ur *mocks.MockUserRepository) {
				ev.EXPECT().FindActiveByTokenHash(gomock.Any(), tokenHash).
					Return(&entity.EmailVerification{
						ID:        verifID,
						UserID:    userID,
						TokenHash: tokenHash,
						ExpiresAt: time.Now().Add(1 * time.Hour),
					}, nil)
				ev.EXPECT().MarkAsVerified(gomock.Any(), verifID).Return(nil)
				ur.EXPECT().VerifyEmail(gomock.Any(), userID).Return(nil)
			},
			wantErr: nil,
		},
		{
			name:  "repo lookup error → wrapped error",
			token: rawToken,
			setupMock: func(ev *mocks.MockEmailVerificationRepository, _ *mocks.MockUserRepository) {
				ev.EXPECT().FindActiveByTokenHash(gomock.Any(), tokenHash).Return(nil, errors.New("db down"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ev := mocks.NewMockEmailVerificationRepository(ctrl)
			ur := mocks.NewMockUserRepository(ctrl)
			tt.setupMock(ev, ur)

			uc := usecase.NewVerifyEmailUseCase(ev, ur)
			err := uc.Execute(context.Background(), tt.token)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			if tt.name == "repo lookup error → wrapped error" {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}
