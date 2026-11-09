package dto

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required"`
}

type LoginResponse struct {
	Token     string `json:"token"`
	TokenType string `json:"token_type"`
	ExpiresAt string `json:"expires_at"`
}
