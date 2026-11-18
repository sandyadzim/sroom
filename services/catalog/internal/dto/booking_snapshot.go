package dto

type BookingSnapshot struct {
	VariantID        string   `json:"variant_id"`
	VehicleLabel     string   `json:"vehicle_label"`
	DailyRateIDR     int64    `json:"daily_rate_idr"`
	Status           string   `json:"status"`
	EligibleUnitIDs  []string `json:"eligible_unit_ids"`
	DemoHoldsEnabled bool     `json:"demo_holds_enabled"`
}
