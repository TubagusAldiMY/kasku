package dto

import (
	"encoding/json"
	"time"
)

// AuditLogItem adalah baris satu entry di list audit log.
type AuditLogItem struct {
	ID           string          `json:"id"`
	AdminID      string          `json:"admin_id"`
	Action       string          `json:"action"`
	TargetUserID *string         `json:"target_user_id,omitempty"`
	TargetEntity *string         `json:"target_entity,omitempty"`
	Metadata     json.RawMessage `json:"metadata"`
	Success      bool            `json:"success"`
	CreatedAt    time.Time       `json:"created_at"`
}
