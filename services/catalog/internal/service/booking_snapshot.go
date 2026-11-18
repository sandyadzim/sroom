package service

import (
	"context"
	"strings"

	"catalog/internal/dto"
)

func (s *CarService) BookingSnapshot(ctx context.Context, id string) (dto.BookingSnapshot, error) {
	car, unitIDs, err := s.repo.BookingSnapshot(ctx, id)
	if err != nil {
		return dto.BookingSnapshot{}, err
	}
	return dto.BookingSnapshot{
		VariantID:        car.ID,
		VehicleLabel:     strings.TrimSpace(car.Brand.Name + " " + car.Model + " " + car.Trim),
		DailyRateIDR:     car.DailyRateIDR,
		Status:           car.Status,
		EligibleUnitIDs:  unitIDs,
		DemoHoldsEnabled: s.repo.DemoHoldsEnabled(),
	}, nil
}
