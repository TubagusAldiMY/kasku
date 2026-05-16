package usecase_test

import (
	"context"
	"errors"
	"testing"

	domainerrors "github.com/TubagusAldiMY/kasku/auth-service/internal/domain/errors"
	"github.com/TubagusAldiMY/kasku/auth-service/internal/usecase"
	"github.com/TubagusAldiMY/kasku/auth-service/tests/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// TestRegisterUseCase_Execute_ValidationFailFast memverifikasi bahwa
// validasi input gagal lebih dulu — sebelum akses DB atau publisher. Ini bisa
// di-test dengan pool nil karena pool tidak akan diakses (early return).
//
// Path happy + DB transaction di-cover di Phase 3 integration test (testcontainer).
func TestRegisterUseCase_Execute_ValidationFailFast(t *testing.T) {
	t.Parallel()

	argon2Cfg := usecase.Argon2Config{Time: 1, MemoryKB: 8 * 1024, Threads: 2, KeyLength: 32}

	tests := []struct {
		name    string
		input   usecase.RegisterInput
		wantErr error // domain sentinel yang harus muncul via errors.Is
	}{
		{
			name:    "invalid email format → ErrValidation",
			input:   usecase.RegisterInput{Email: "not-an-email", Username: "alice", Password: "Pass1234"},
			wantErr: domainerrors.ErrValidation,
		},
		{
			name:    "empty email → ErrValidation",
			input:   usecase.RegisterInput{Email: "", Username: "alice", Password: "Pass1234"},
			wantErr: domainerrors.ErrValidation,
		},
		{
			name:    "username too short (< 3 char) → ErrValidation",
			input:   usecase.RegisterInput{Email: "u@ex.com", Username: "ab", Password: "Pass1234"},
			wantErr: domainerrors.ErrValidation,
		},
		{
			name:    "username too long (> 30 char) → ErrValidation",
			input:   usecase.RegisterInput{Email: "u@ex.com", Username: "thisusernameiswaytoolongtobevalidanymore", Password: "Pass1234"},
			wantErr: domainerrors.ErrValidation,
		},
		{
			name:    "username contains invalid char → ErrValidation",
			input:   usecase.RegisterInput{Email: "u@ex.com", Username: "with space", Password: "Pass1234"},
			wantErr: domainerrors.ErrValidation,
		},
		{
			name:    "password too short → ErrPasswordTooShort",
			input:   usecase.RegisterInput{Email: "u@ex.com", Username: "alice", Password: "Sh0rt"},
			wantErr: domainerrors.ErrPasswordTooShort,
		},
		{
			name:    "password no uppercase → ErrPasswordTooWeak",
			input:   usecase.RegisterInput{Email: "u@ex.com", Username: "alice", Password: "lowercase1"},
			wantErr: domainerrors.ErrPasswordTooWeak,
		},
		{
			name:    "password no digit → ErrPasswordTooWeak",
			input:   usecase.RegisterInput{Email: "u@ex.com", Username: "alice", Password: "NoDigitHere"},
			wantErr: domainerrors.ErrPasswordTooWeak,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ur := mocks.NewMockUserRepository(ctrl) // no expectations — validation must fail first
			pub := mocks.NewMockEventPublisher(ctrl)

			// pool nil aman: validateRegisterInput gagal → early return sebelum pool diakses
			uc := usecase.NewRegisterUseCase(nil, ur, pub, argon2Cfg)
			_, err := uc.Execute(context.Background(), tt.input)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

// TestRegisterUseCase_Execute_DuplicateEmail menguji jalur yang gagal pada cek
// ExistsByEmail — masih tidak butuh pool (gagal sebelum tx.Begin).
func TestRegisterUseCase_Execute_DuplicateEmail(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ur := mocks.NewMockUserRepository(ctrl)
	pub := mocks.NewMockEventPublisher(ctrl)
	ur.EXPECT().ExistsByEmail(gomock.Any(), "dup@ex.com").Return(true, nil)

	argon2Cfg := usecase.Argon2Config{Time: 1, MemoryKB: 8 * 1024, Threads: 2, KeyLength: 32}
	uc := usecase.NewRegisterUseCase(nil, ur, pub, argon2Cfg)
	_, err := uc.Execute(context.Background(), usecase.RegisterInput{
		Email: "dup@ex.com", Username: "alice", Password: "Pass1234",
	})
	assert.ErrorIs(t, err, domainerrors.ErrEmailAlreadyExists)
}

func TestRegisterUseCase_Execute_DuplicateUsername(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ur := mocks.NewMockUserRepository(ctrl)
	pub := mocks.NewMockEventPublisher(ctrl)
	ur.EXPECT().ExistsByEmail(gomock.Any(), "u@ex.com").Return(false, nil)
	ur.EXPECT().ExistsByUsername(gomock.Any(), "alice").Return(true, nil)

	argon2Cfg := usecase.Argon2Config{Time: 1, MemoryKB: 8 * 1024, Threads: 2, KeyLength: 32}
	uc := usecase.NewRegisterUseCase(nil, ur, pub, argon2Cfg)
	_, err := uc.Execute(context.Background(), usecase.RegisterInput{
		Email: "u@ex.com", Username: "alice", Password: "Pass1234",
	})
	assert.ErrorIs(t, err, domainerrors.ErrUsernameAlreadyExists)
}

func TestRegisterUseCase_Execute_RepoLookupError(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ur := mocks.NewMockUserRepository(ctrl)
	pub := mocks.NewMockEventPublisher(ctrl)
	ur.EXPECT().ExistsByEmail(gomock.Any(), "u@ex.com").Return(false, errors.New("db down"))

	argon2Cfg := usecase.Argon2Config{Time: 1, MemoryKB: 8 * 1024, Threads: 2, KeyLength: 32}
	uc := usecase.NewRegisterUseCase(nil, ur, pub, argon2Cfg)
	_, err := uc.Execute(context.Background(), usecase.RegisterInput{
		Email: "u@ex.com", Username: "alice", Password: "Pass1234",
	})
	assert.Error(t, err)
}

func TestValidatePassword(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		pass    string
		wantErr error
	}{
		{"too short", "Ab1", domainerrors.ErrPasswordTooShort},
		{"no upper", "lowercase1", domainerrors.ErrPasswordTooWeak},
		{"no lower", "UPPERCASE1", domainerrors.ErrPasswordTooWeak},
		{"no digit", "NoDigitOnly", domainerrors.ErrPasswordTooWeak},
		{"valid", "Pass1234", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := usecase.ValidatePasswordForTest(tt.pass)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)
		})
	}
}
