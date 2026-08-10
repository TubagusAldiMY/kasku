package errors

import "fmt"

type DomainError struct {
	Code    string
	Message string
}

func (e *DomainError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

var (
	ErrUserNotFound = &DomainError{Code: "USER_NOT_FOUND", Message: "User tidak ditemukan."}
	ErrInternal     = &DomainError{Code: "INTERNAL_ERROR", Message: "Terjadi kesalahan internal."}

	// ErrUsernameTaken dipisahkan dari kegagalan umum supaya handler bisa
	// menjawab 409 dengan sebab yang jelas. Sebelumnya semua kegagalan rename
	// jatuh ke satu pesan "Gagal memperbarui profil" — user tidak punya cara
	// tahu bahwa yang perlu diubah hanyalah pilihan namanya.
	ErrUsernameTaken = &DomainError{Code: "USERNAME_TAKEN", Message: "Username sudah dipakai. Pilih yang lain."}

	// ErrUsernameInvalid menandai format username yang tidak memenuhi aturan.
	ErrUsernameInvalid = &DomainError{
		Code:    "USERNAME_INVALID",
		Message: "Username harus 3-30 karakter dan hanya boleh berisi huruf, angka, titik, garis bawah, atau strip.",
	}

	// ErrAvatarInvalid menandai berkas unggahan yang bukan gambar yang didukung
	// atau melebihi batas ukuran.
	ErrAvatarInvalid = &DomainError{
		Code:    "AVATAR_INVALID",
		Message: "Foto harus berupa JPEG, PNG, atau WebP berukuran maksimal 5MB.",
	}

	// ErrAvatarNotFound dipakai saat user belum pernah mengunggah foto.
	ErrAvatarNotFound = &DomainError{Code: "AVATAR_NOT_FOUND", Message: "Foto profil belum ada."}
)

func IsDomainError(err error) bool {
	_, ok := err.(*DomainError)
	return ok
}
