package service

import (
	"context"
	"time"

	"catalog/internal/dto"
)

func (s *CarService) FleetSummary(ctx context.Context) (dto.FleetSummary, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return s.repo.FleetSummary(ctx)
}
