package dto

import "identity/internal/identity/model"

type MeResponse struct {
	ID        string     `json:"id"`
	FullName  string     `json:"full_name"`
	Email     string     `json:"email"`
	Role      model.Role `json:"role"`
	CreatedAt string     `json:"created_at"`
}
