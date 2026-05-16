package grpc

import (
	"context"
	"errors"
	"fmt"
	"testing"

	domainerrors "github.com/TubagusAldiMY/kasku/billing-service/internal/domain/errors"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestToStatus(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want codes.Code
	}{
		{"nil → nil status", nil, codes.OK},
		{"context.DeadlineExceeded → DeadlineExceeded", context.DeadlineExceeded, codes.DeadlineExceeded},
		{"context.Canceled → Canceled", context.Canceled, codes.Canceled},
		{"pgx.ErrNoRows → NotFound", pgx.ErrNoRows, codes.NotFound},
		{"ErrSubscriptionNotFound → NotFound", domainerrors.ErrSubscriptionNotFound, codes.NotFound},
		{"ErrPlanNotFound → NotFound", domainerrors.ErrPlanNotFound, codes.NotFound},
		{"wrapped pgx.ErrNoRows → NotFound", fmt.Errorf("wrap: %w", pgx.ErrNoRows), codes.NotFound},
		{"unknown error → Internal", errors.New("random"), codes.Internal},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := toStatus(tc.err)
			if tc.err == nil {
				assert.Nil(t, got)
				return
			}
			st, ok := status.FromError(got)
			assert.True(t, ok)
			assert.Equal(t, tc.want, st.Code())
		})
	}
}
