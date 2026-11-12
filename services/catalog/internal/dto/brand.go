package dto

import "encoding/json"

type CreateBrandRequest struct {
	Name           string  `json:"name" validate:"required,max=100"`
	ImageObjectKey *string `json:"image_object_key"`
}

type UpdateBrandRequest struct {
	Name           json.RawMessage `json:"name"`
	ImageObjectKey json.RawMessage `json:"image_object_key"`
	Status         json.RawMessage `json:"status"`
}

type BrandResponse struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	ImageObjectKey *string `json:"image_object_key"`
	Status         string  `json:"status"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

type BrandListResponse struct {
	Data []BrandResponse `json:"data"`
	Page BrandPage       `json:"page"`
}

type BrandPage struct {
	Number int   `json:"number"`
	Limit  int   `json:"limit"`
	Total  int64 `json:"total"`
}
