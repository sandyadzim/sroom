package dto

import "identity/internal/identity/model"

type CreateAdminRequest struct {
	FullName string  `json:"full_name" validate:"max=100"`
	Email    string  `json:"email" validate:"required,email,max=255"`
	Password string  `json:"password" validate:"required,min=8"`
	Status   *string `json:"status" validate:"omitempty,oneof=active inactive"`
}

type UpdateAdminRequest struct {
	FullName *string `json:"full_name" validate:"omitempty,max=100"`
	Email    *string `json:"email" validate:"omitempty,email,max=255"`
	Password *string `json:"password" validate:"omitempty,min=8"`
	Status   *string `json:"status" validate:"omitempty,oneof=active inactive"`
}

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
	Page AccountPage     `json:"page"`
}

type AccountPage struct {
	Number int   `json:"number"`
	Limit  int   `json:"limit"`
	Total  int64 `json:"total"`
}
