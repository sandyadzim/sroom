package dto

type UpdateFeaturedCarsRequest struct {
	CarIDs []*string `json:"car_ids"`
}

type AdminFeaturedCar struct {
	CarCard
	Position int `json:"position"`
	Status string `json:"status"`
}
