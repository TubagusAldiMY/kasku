package usecase

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

// randReadFull mengisi slice dengan random bytes yang kriptografis aman.
// Mengembalikan error jika sumber entropy tidak tersedia (keadaan kritis).
func randReadFull(buf []byte) (int, error) {
	n, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		return 0, fmt.Errorf("gagal baca random bytes: %w", err)
	}
	return n, nil
}

// base64Encode mengencoding bytes ke base64 Raw Standard Encoding (tanpa padding).
func base64Encode(data []byte) string {
	return base64.RawStdEncoding.EncodeToString(data)
}
