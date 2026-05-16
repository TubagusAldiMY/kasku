package usecase_test

import (
	"context"
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

func TestResendVerificationUseCase_Execute(t *testing.T) {
	t.Parallel()

	t.Run("happy: invalidate old + create new + publish", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ur := mocks.NewMockUserRepository(ctrl)
		ev := mocks.NewMockEmailVerificationRepository(ctrl)
		pub := mocks.NewMockEventPublisher(ctrl)

		userID := uuid.New()
		user := &entity.User{
			ID:            userID,
			Email:         "u@example.com",
			EmailVerified: false,
		}
		ur.EXPECT().FindByEmail(gomock.Any(), "u@example.com").Return(user, nil)
		ev.EXPECT().InvalidateAllActiveByUserID(gomock.Any(), userID).Return(nil)
		ev.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
		pub.EXPECT().PublishEmailVerificationResent(gomock.Any(), gomock.Any()).Return(nil)

		uc := usecase.NewResendVerificationUseCase(ur, ev, pub, nil, usecase.EmailRateLimit{})
		assert.NoError(t, uc.Execute(context.Background(), "u@example.com"))
	})

	t.Run("email already verified: silent, no publish", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ur := mocks.NewMockUserRepository(ctrl)
		ev := mocks.NewMockEmailVerificationRepository(ctrl)
		pub := mocks.NewMockEventPublisher(ctrl)

		user := &entity.User{
			ID:            uuid.New(),
			Email:         "v@example.com",
			EmailVerified: true,
		}
		ur.EXPECT().FindByEmail(gomock.Any(), "v@example.com").Return(user, nil)

		uc := usecase.NewResendVerificationUseCase(ur, ev, pub, nil, usecase.EmailRateLimit{})
		assert.NoError(t, uc.Execute(context.Background(), "v@example.com"))
	})

	t.Run("user not found: silent (anti-enumeration)", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ur := mocks.NewMockUserRepository(ctrl)
		ev := mocks.NewMockEmailVerificationRepository(ctrl)
		pub := mocks.NewMockEventPublisher(ctrl)

		ur.EXPECT().FindByEmail(gomock.Any(), "x@example.com").Return(nil, nil)

		uc := usecase.NewResendVerificationUseCase(ur, ev, pub, nil, usecase.EmailRateLimit{})
		assert.NoError(t, uc.Execute(context.Background(), "x@example.com"))
	})

	t.Run("rate-limit exceeded → silent, no DB call", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		ur := mocks.NewMockUserRepository(ctrl)
		ev := mocks.NewMockEmailVerificationRepository(ctrl)
		pub := mocks.NewMockEventPublisher(ctrl)
		lim := mocks.NewMockLimiter(ctrl)

		lim.EXPECT().Check(gomock.Any(), gomock.Any(), 3, time.Hour).
			Return(time.Duration(0), ratelimit.ErrLimitExceeded)

		uc := usecase.NewResendVerificationUseCase(ur, ev, pub, lim,
			usecase.EmailRateLimit{Limit: 3, Window: time.Hour, Endpoint: "resend:email"})
		assert.NoError(t, uc.Execute(context.Background(), "a@b.com"))
	})
}
