package dto

import "identity/internal/identity/model"

type AdminResponse struct {
	ID        string     `json:"id"`
	FullName  string     `json:"full_name"`
	Email     string     `json:"email"`
	Role      model.Role `json:"role"`
	Status    string     `json:"status"`
	CreatedAt string     `json:"created_at"`
	UpdatedAt string     `json:"updated_at"`
}

type AdminListResponse struct {
	Data []AdminResponse `json:"data"`
	Page AdminPage       `json:"page"`
}

type AdminPage struct {
	Number int   `json:"number"`
	Limit  int   `json:"limit"`
	Total  int64 `json:"total"`
}
