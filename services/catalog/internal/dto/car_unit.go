package dto

type CreateCarUnitRequest struct {
	InternalCode string  `json:"internal_code"`
	Status       *string `json:"status"`
}

type UpdateCarUnitRequest struct {
	InternalCode *string `json:"internal_code"`
	Status       *string `json:"status"`
}

type CarUnitResponse struct {
	ID           string `json:"id"`
	CarID        string `json:"car_id"`
	InternalCode string `json:"internal_code"`
	Status       string `json:"status"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type CarUnitListResponse struct {
	Data []CarUnitResponse `json:"data"`
	Page CarPage           `json:"page"`
}
