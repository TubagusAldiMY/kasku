package usecase_test

import (
	"context"
	"testing"

	"github.com/TubagusAldiMY/kasku/admin-service/internal/usecase"
	"github.com/TubagusAldiMY/kasku/admin-service/tests/mocks"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// TestSeedBootstrapAdmin_WeakPassword memastikan startup GAGAL (dan tidak ada admin
// yang dibuat) untuk password placeholder maupun terlalu pendek.
func TestSeedBootstrapAdmin_WeakPassword(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"placeholder .env.example": "ChangeMe-Strong-Passw0rd!",
		"too short":                "short1!",
		"empty":                    "",
	}

	for name, pw := range cases {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockAdminUserRepository(ctrl)
			// Tabel kosong → guard harus jalan sebelum CreateBootstrap dipanggil.
			repo.EXPECT().Count(gomock.Any()).Return(int64(0), nil)
			// CreateBootstrap TIDAK boleh dipanggil (gomock akan gagal bila terpanggil).

			err := usecase.SeedBootstrapAdmin(context.Background(), repo, usecase.BootstrapInput{
				Username: "superadmin",
				Password: pw,
				Argon2:   testArgon2(),
			}, zerolog.Nop())
			require.Error(t, err)
		})
	}
}

// TestSeedBootstrapAdmin_StrongPassword memastikan password kuat lolos dan admin dibuat.
func TestSeedBootstrapAdmin_StrongPassword(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	repo := mocks.NewMockAdminUserRepository(ctrl)
	repo.EXPECT().Count(gomock.Any()).Return(int64(0), nil)
	repo.EXPECT().CreateBootstrap(gomock.Any(), gomock.Any()).Return(nil)

	err := usecase.SeedBootstrapAdmin(context.Background(), repo, usecase.BootstrapInput{
		Username: "superadmin",
		Password: "aVeryStr0ng-Bootstrap-Pass!",
		Argon2:   testArgon2(),
	}, zerolog.Nop())
	require.NoError(t, err)
}
