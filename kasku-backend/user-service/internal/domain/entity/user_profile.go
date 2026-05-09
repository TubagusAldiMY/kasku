package entity

import (
	"time"

	"github.com/google/uuid"
)

// UserProfile merepresentasikan profil user yang ditampilkan ke client.
// Data ini dibaca dari JWT claims (headers dari api-gateway), bukan dari DB.
type UserProfile struct {
	UserID           uuid.UUID
	TenantSchema     string
	SubscriptionTier string
}

// ProvisioningRequest merupakan data untuk provisioning tenant baru.
type ProvisioningRequest struct {
	UserID    uuid.UUID
	Email     string
	Username  string
	CreatedAt time.Time
}
