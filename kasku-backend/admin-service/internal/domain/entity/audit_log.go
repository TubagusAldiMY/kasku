package entity

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AuditAction adalah enum untuk semua aksi admin yang tercatat.
type AuditAction string

const (
	AuditActionLogin                AuditAction = "LOGIN"
	AuditActionLogout               AuditAction = "LOGOUT"
	AuditActionSuspendUser          AuditAction = "SUSPEND_USER"
	AuditActionActivateUser         AuditAction = "ACTIVATE_USER"
	AuditActionOverrideSubscription AuditAction = "OVERRIDE_SUBSCRIPTION"
)

// AuditLogEntry adalah catatan satu aksi admin.
// metadata menampung detail kontekstual (reason, old/new tier, dll) dalam JSON.
type AuditLogEntry struct {
	ID            uuid.UUID
	AdminID       uuid.UUID
	Action        AuditAction
	TargetUserID  *uuid.UUID
	TargetEntity  *string
	Metadata      json.RawMessage
	Success       bool
	CreatedAt     time.Time
}
