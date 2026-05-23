package usecase

import (
	"fmt"

	"github.com/google/uuid"
)

// parseUUIDFromString mem-parse string menjadi uuid.UUID dengan error yang deskriptif.
func parseUUIDFromString(raw string) (uuid.UUID, error) {
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, fmt.Errorf("nilai %q bukan UUID yang valid: %w", raw, err)
	}
	return parsed, nil
}
