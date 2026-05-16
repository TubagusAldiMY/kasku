package dto

import "time"

// LoginRequest payload login admin.
type LoginRequest struct {
	Username string `json:"username" binding:"required,min=3,max=30"`
	Password string `json:"password" binding:"required,min=8"`
}

// LoginResponse menjawab login admin sukses.
type LoginResponse struct {
	AccessToken string         `json:"access_token"`
	TokenType   string         `json:"token_type"`
	ExpiresIn   int64          `json:"expires_in"`
	Admin       AdminUserDTO   `json:"admin"`
}

// AdminUserDTO adalah representasi admin profile.
type AdminUserDTO struct {
	ID          string     `json:"id"`
	Username    string     `json:"username"`
	Role        string     `json:"role"`
	IsActive    bool       `json:"is_active"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}
