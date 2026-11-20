package dto

type FleetCounts struct {
	Total    int64 `json:"total"`
	Active   int64 `json:"active"`
	Inactive int64 `json:"inactive"`
}

type FleetSummary struct {
	Cars  FleetCounts `json:"cars"`
	Units FleetCounts `json:"units"`
}
