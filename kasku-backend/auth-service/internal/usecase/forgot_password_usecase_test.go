package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/TubagusAldiMY/kasku/auth-service/internal/domain/entity"
	"github.com/TubagusAldiMY/kasku/auth-service/internal/infrastructure/ratelimit"
	"github.com/TubagusAldiMY/kasku/auth-service/internal/usecase"
	"github.com/TubagusAldiMY/kasku/auth-service/tests/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestForgotPasswordUseCase_Execute_EnumerationPrevention(t *testing.T) {
	t.Parallel()

	// Critical security property: response identik untuk email yang ada vs tidak ada.
	// Caller (handler) selalu return 200 OK dengan message generic.
	// Use case selalu return nil pada path normal.

	tests := []struct {
		name      string
		email     string
		setupMock func(ur *mocks.MockUserRepository, rr *mocks.MockPasswordResetRepository, pub *mocks.MockEventPublisher)
	}{
		{
			name:  "email exists + active: publish event",
			email: "exists@example.com",
			setupMock: func(ur *mocks.MockUserRepository, rr *mocks.MockPasswordResetRepository, pub *mocks.MockEventPublisher) {
				user := &entity.User{
					ID:       uuid.New(),
					Email:    "exists@example.com",
					IsActive: true,
				}
				ur.EXPECT().FindByEmail(gomock.Any(), "exists@example.com").Return(user, nil)
				rr.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
				pub.EXPECT().PublishPasswordResetRequested(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
		{
			name:  "email NOT found: identical observable behavior (no publish)",
			email: "missing@example.com",
			setupMock: func(ur *mocks.MockUserRepository, _ *mocks.MockPasswordResetRepository, _ *mocks.MockEventPublisher) {
				ur.EXPECT().FindByEmail(gomock.Any(), "missing@example.com").Return(nil, nil)
			},
		},
		{
			name:  "email exists but INACTIVE: silent (no publish)",
			email: "inactive@example.com",
			setupMock: func(ur *mocks.MockUserRepository, _ *mocks.MockPasswordResetRepository, _ *mocks.MockEventPublisher) {
				user := &entity.User{
					ID:       uuid.New(),
					Email:    "inactive@example.com",
					IsActive: false,
				}
				ur.EXPECT().FindByEmail(gomock.Any(), "inactive@example.com").Return(user, nil)
			},
		},
		{
			name:  "lookup infra error: silent (anti-enumeration)",
			email: "any@example.com",
			setupMock: func(ur *mocks.MockUserRepository, _ *mocks.MockPasswordResetRepository, _ *mocks.MockEventPublisher) {
				ur.EXPECT().FindByEmail(gomock.Any(), "any@example.com").Return(nil, errors.New("db down"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ur := mocks.NewMockUserRepository(ctrl)
			rr := mocks.NewMockPasswordResetRepository(ctrl)
			pub := mocks.NewMockEventPublisher(ctrl)
			tt.setupMock(ur, rr, pub)

			// limiter nil → bypass rate-limit untuk fokus test enumeration
			uc := usecase.NewForgotPasswordUseCase(ur, rr, pub, nil, usecase.EmailRateLimit{})
			err := uc.Execute(context.Background(), tt.email)
			// CRITICAL: semua path return nil → identical observable behavior
			assert.NoError(t, err)
		})
	}
}

func TestForgotPasswordUseCase_Execute_RateLimit(t *testing.T) {
	t.Parallel()

	t.Run("rate-limit exceeded → silent, no DB call", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ur := mocks.NewMockUserRepository(ctrl)
		rr := mocks.NewMockPasswordResetRepository(ctrl)
		pub := mocks.NewMockEventPublisher(ctrl)
		lim := mocks.NewMockLimiter(ctrl)

		lim.EXPECT().Check(gomock.Any(), gomock.Any(), 3, time.Hour).
			Return(time.Duration(0), ratelimit.ErrLimitExceeded)
		// no DB lookup, no publish — fully short-circuit

		uc := usecase.NewForgotPasswordUseCase(ur, rr, pub, lim,
			usecase.EmailRateLimit{Limit: 3, Window: time.Hour, Endpoint: "forgot:email"})
		err := uc.Execute(context.Background(), "a@b.com")
		assert.NoError(t, err)
	})

	t.Run("rate-limit infra error → fail-open (DB call proceeds)", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ur := mocks.NewMockUserRepository(ctrl)
		rr := mocks.NewMockPasswordResetRepository(ctrl)
		pub := mocks.NewMockEventPublisher(ctrl)
		lim := mocks.NewMockLimiter(ctrl)

		lim.EXPECT().Check(gomock.Any(), gomock.Any(), 3, time.Hour).
			Return(time.Duration(0), errors.New("redis down"))
		ur.EXPECT().FindByEmail(gomock.Any(), "a@b.com").Return(nil, nil)

		uc := usecase.NewForgotPasswordUseCase(ur, rr, pub, lim,
			usecase.EmailRateLimit{Limit: 3, Window: time.Hour, Endpoint: "forgot:email"})
		err := uc.Execute(context.Background(), "a@b.com")
		assert.NoError(t, err)
	})
}
