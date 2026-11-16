package dto

type UpdateBannerRequest struct {
	CarID          string `json:"car_id,omitempty" validate:"omitempty,uuid"`
	BrandID        string `json:"brand_id,omitempty" validate:"omitempty,uuid"`
	CarModel       string `json:"car_model,omitempty" validate:"max=100"`
	CarSlug        string `json:"car_slug,omitempty" validate:"max=150"`
	HeadlineLine1  string `json:"headline_line_1" validate:"required,max=100"`
	HeadlineLine2  string `json:"headline_line_2" validate:"required,max=100"`
	Description    string `json:"description" validate:"required,max=500"`
	ImageObjectKey string `json:"image_object_key" validate:"required"`
}

type BannerResponse struct {
	Position int            `json:"position"`
	Content  *BannerContent `json:"content"`
}

type BannerContent struct {
	CarID          *string `json:"car_id"`
	BrandID        string  `json:"brand_id"`
	BrandName      string  `json:"brand_name"`
	CarModel       string  `json:"car_model"`
	CarSlug        string  `json:"car_slug"`
	HeadlineLine1  string  `json:"headline_line_1"`
	HeadlineLine2  string  `json:"headline_line_2"`
	Description    string  `json:"description"`
	ImageObjectKey string  `json:"image_object_key"`
	ImageURL       *string `json:"image_url"`
}

type PublicBannerResponse struct {
	Slug        string    `json:"slug"`
	Make        string    `json:"make"`
	Model       string    `json:"model"`
	Image       *string   `json:"image"`
	Headline    [2]string `json:"headline"`
	Description string    `json:"description"`
}
