package dto

type UpdateFeaturedCarsRequest struct {
	CarIDs []string `json:"car_ids"`
}

type AdminFeaturedCar struct {
	CarCard
	Status string `json:"status"`
}
