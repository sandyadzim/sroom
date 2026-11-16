package dto

type CreateCarRequest struct {
	BrandID      string          `json:"brand_id" validate:"required,uuid"`
	Model        string          `json:"model" validate:"required,max=100"`
	Trim         string          `json:"trim" validate:"max=100"`
	Category     string          `json:"category" validate:"oneof=economy city mpv suv premium"`
	BodyType     string          `json:"body_type" validate:"oneof=hatchback sedan mpv suv"`
	Seats        int             `json:"seats" validate:"min=1,max=2147483647"`
	Transmission string          `json:"transmission" validate:"oneof=automatic manual"`
	FuelType     string          `json:"fuel_type" validate:"oneof=petrol diesel hybrid electric"`
	Description  string          `json:"description" validate:"required,max=2000"`
	DailyRateIDR int64           `json:"daily_rate_idr" validate:"gt=0"`
	Status       *string         `json:"status,omitempty" validate:"omitempty,oneof=active inactive"`
	Cover        *CarPhotoInput  `json:"cover" validate:"required"`
	Gallery      []CarPhotoInput `json:"gallery" validate:"min=1"`
}

type UpdateCarRequest struct {
	BrandID      *string         `json:"brand_id" validate:"omitempty,uuid"`
	Model        *string         `json:"model" validate:"omitempty,min=1,max=100"`
	Trim         *string         `json:"trim" validate:"omitempty,max=100"`
	Category     *string         `json:"category" validate:"omitempty,oneof=economy city mpv suv premium"`
	BodyType     *string         `json:"body_type" validate:"omitempty,oneof=hatchback sedan mpv suv"`
	Seats        *int            `json:"seats" validate:"omitempty,min=1,max=2147483647"`
	Transmission *string         `json:"transmission" validate:"omitempty,oneof=automatic manual"`
	FuelType     *string         `json:"fuel_type" validate:"omitempty,oneof=petrol diesel hybrid electric"`
	Description  *string         `json:"description" validate:"omitempty,min=1,max=2000"`
	DailyRateIDR *int64          `json:"daily_rate_idr" validate:"omitempty,gt=0"`
	Status       *string         `json:"status" validate:"omitempty,oneof=active inactive"`
	Cover        *CarPhotoInput  `json:"cover"`
	Gallery      []CarPhotoInput `json:"gallery" validate:"omitempty,min=1"`
}

type CarPhotoInput struct {
	ObjectKey string `json:"object_key" validate:"required"`
	AltText   string `json:"alt_text" validate:"required,max=200"`
	Label     string `json:"label" validate:"required,max=100"`
}

type CarImageResponse struct {
	URL     *string `json:"url"`
	AltText string  `json:"alt_text"`
	Label   string  `json:"label"`
}

type AdminCarImageResponse struct {
	CarImageResponse
	ID        string `json:"id"`
	ObjectKey string `json:"object_key"`
	Kind      string `json:"kind"`
}

type CarCard struct {
	ID           string            `json:"id"`
	BrandID      string            `json:"brand_id"`
	BrandName    string            `json:"brand_name"`
	Model        string            `json:"model"`
	Trim         string            `json:"trim"`
	Slug         string            `json:"slug"`
	Category     string            `json:"category"`
	Seats        int               `json:"seats"`
	Transmission string            `json:"transmission"`
	DailyRateIDR int64             `json:"daily_rate_idr"`
	Cover        *CarImageResponse `json:"cover"`
}

type CarDetail struct {
	CarCard
	Status      string             `json:"status"`
	BodyType    string             `json:"body_type"`
	FuelType    string             `json:"fuel_type"`
	Description string             `json:"description"`
	Gallery     []CarImageResponse `json:"gallery"`
}

type AdminCarCard struct {
	CarCard
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type AdminCarDetail struct {
	AdminCarCard
	BodyType    string                  `json:"body_type"`
	FuelType    string                  `json:"fuel_type"`
	Description string                  `json:"description"`
	Images      []AdminCarImageResponse `json:"images"`
}

type CarPage struct {
	Number int   `json:"number"`
	Limit  int   `json:"limit"`
	Total  int64 `json:"total"`
}

type CarList struct {
	Data []CarCard `json:"data"`
	Page CarPage   `json:"page"`
}

type AdminCarList struct {
	Data []AdminCarCard `json:"data"`
	Page CarPage        `json:"page"`
}

type CarBrandOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
