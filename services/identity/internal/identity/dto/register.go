package dto

type RegisterRequest struct {
    FullName string `json:"full_name" validate:"max=100"`
    Email    string `json:"email" validate:"required,email,max=255"`
    Password string `json:"password" validate:"required,min=8"`
}

type RegisterResponse struct {
    ID       string `json:"id"`
    FullName string `json:"full_name"`
    Email    string `json:"email"`
}
