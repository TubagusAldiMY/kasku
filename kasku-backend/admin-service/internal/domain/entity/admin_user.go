package entity

import (
	"time"

	"github.com/google/uuid"
)

// AdminRole adalah role admin yang menentukan otorisasi.
type AdminRole string

const (
	AdminRoleSuperAdmin AdminRole = "SUPER_ADMIN"
	AdminRoleSupport    AdminRole = "SUPPORT"
)

// IsValid mengembalikan true bila role termasuk yang didefinisikan.
func (r AdminRole) IsValid() bool {
	switch r {
	case AdminRoleSuperAdmin, AdminRoleSupport:
		return true
	default:
		return false
	}
}

// AdminUser merepresentasikan operator dashboard admin.
// Disimpan di kasku_admin.admin_users (terisolasi dari kasku_auth.users).
type AdminUser struct {
	ID           uuid.UUID
	Username     string
	PasswordHash string
	Role         AdminRole
	IsActive     bool
	LastLoginAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// IsSuperAdmin mengembalikan true bila admin memiliki role tertinggi.
func (a *AdminUser) IsSuperAdmin() bool {
	return a.Role == AdminRoleSuperAdmin
}
