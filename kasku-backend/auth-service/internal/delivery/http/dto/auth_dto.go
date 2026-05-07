package dto

// RegisterRequest adalah body request untuk POST /auth/register.
type RegisterRequest struct {
	Email    string `json:"email"    binding:"required"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RegisterResponse dikembalikan saat registrasi berhasil.
type RegisterResponse struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Message  string `json:"message"`
}

// LoginRequest adalah body request untuk POST /auth/login.
type LoginRequest struct {
	Email    string `json:"email"    binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse dikembalikan saat login berhasil.
type LoginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

// ResendVerificationRequest adalah body request untuk POST /auth/resend-verification.
type ResendVerificationRequest struct {
	Email string `json:"email" binding:"required"`
}

// ForgotPasswordRequest adalah body request untuk POST /auth/forgot-password.
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required"`
}

// ResetPasswordRequest adalah body request untuk POST /auth/reset-password.
type ResetPasswordRequest struct {
	Token       string `json:"token"        binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

// HealthDependency merepresentasikan status satu dependency.
type HealthDependency struct {
	Status string `json:"status"`
}

// HealthResponse dikembalikan oleh GET /health.
type HealthResponse struct {
	Status       string                      `json:"status"`
	Service      string                      `json:"service"`
	Version      string                      `json:"version"`
	Dependencies map[string]HealthDependency `json:"dependencies"`
}
