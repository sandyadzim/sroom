package dto

type UpdateCustomerStatusRequest struct {
	Status string `json:"status"`
}

type CustomerResponse struct {
	ID        string `json:"id"`
	FullName  string `json:"full_name"`
	Email     string `json:"email"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type CustomerListResponse struct {
	Data []CustomerResponse `json:"data"`
	Page AccountPage        `json:"page"`
}
