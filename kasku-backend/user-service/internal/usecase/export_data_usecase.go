package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/TubagusAldiMY/kasku/user-service/internal/infrastructure/persistence"
	"github.com/rs/zerolog"
)

// ExportData adalah bundel lengkap data pribadi user untuk portabilitas (UU PDP / GDPR).
type ExportData struct {
	ExportedAt   time.Time                      `json:"exported_at"`
	Profile      *persistence.UserProfileRecord `json:"profile"`
	Subscription json.RawMessage                `json:"subscription"`
	Finance      map[string]json.RawMessage     `json:"finance"`
	Notice       string                         `json:"notice"`
}

// ExportDataUseCase mengumpulkan profil + langganan + seluruh data keuangan tenant.
type ExportDataUseCase struct {
	profileRepo persistence.UserProfileRepository
	exportRepo  persistence.ExportRepository
	now         func() time.Time
	log         zerolog.Logger
}

func NewExportDataUseCase(
	profileRepo persistence.UserProfileRepository,
	exportRepo persistence.ExportRepository,
	log zerolog.Logger,
) *ExportDataUseCase {
	return &ExportDataUseCase{
		profileRepo: profileRepo,
		exportRepo:  exportRepo,
		now:         time.Now,
		log:         log,
	}
}

func (uc *ExportDataUseCase) Execute(ctx context.Context, userID, tenantSchema string) (*ExportData, error) {
	profile, err := uc.profileRepo.GetUserProfile(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("gagal ambil profil: %w", err)
	}

	finance, err := uc.exportRepo.DumpTenantData(ctx, tenantSchema)
	if err != nil {
		return nil, fmt.Errorf("gagal ambil data keuangan: %w", err)
	}

	subscription, err := uc.exportRepo.DumpSubscription(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("gagal ambil data langganan: %w", err)
	}

	return &ExportData{
		ExportedAt:   uc.now().UTC(),
		Profile:      profile,
		Subscription: subscription,
		Finance:      finance,
		Notice:       "Ekspor ini berisi profil, langganan, dan seluruh data keuangan Anda di KasKu. Metadata autentikasi (mis. waktu login terakhir) dikelola terpisah oleh auth-service dan tidak termasuk di sini.",
	}, nil
}
