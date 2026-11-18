package checkout

import (
	"context"
	"errors"
	"time"

	"booking/internal/catalog"
)

var ErrDatabaseUnavailable = errors.New("booking database is unavailable")

type Quote struct {
	VariantID    string    `json:"variant_id"`
	VehicleLabel string    `json:"vehicle_label"`
	PickupAt     time.Time `json:"pickup_at"`
	ReturnAt     time.Time `json:"return_at"`
	RentalDays   int       `json:"rental_days"`
	DailyRateIDR int64     `json:"daily_rate_idr"`
	TotalIDR     int64     `json:"total_idr"`
	Currency     string    `json:"currency"`
	Available    bool      `json:"available"`
	QuotedAt     time.Time `json:"quoted_at"`
}

type Service struct {
	catalog     *catalog.Client
	allocations *Repository
	wita        *time.Location
}

func NewService(client *catalog.Client, allocations *Repository) (*Service, error) {
	wita, err := time.LoadLocation("Asia/Makassar")
	if err != nil {
		return nil, errors.New("could not load Asia/Makassar time zone")
	}
	return &Service{catalog: client, allocations: allocations, wita: wita}, nil
}

func (s *Service) Quote(ctx context.Context, variantID string, pickup, returnAt, now time.Time, requestID string) (Quote, error) {
	pickup, returnAt = pickup.In(s.wita), returnAt.In(s.wita)
	days, err := rentalDays(pickup, returnAt, now.In(s.wita))
	if err != nil {
		return Quote{}, err
	}
	if s.catalog == nil {
		return Quote{}, catalog.ErrUnavailable
	}
	snapshot, err := s.catalog.BookingSnapshot(ctx, variantID, requestID)
	if err != nil {
		return Quote{}, err
	}
	total, err := quoteTotal(snapshot.DailyRateIDR, days)
	if err != nil {
		return Quote{}, err
	}
	available := false
	if snapshot.Status == "active" && len(snapshot.EligibleUnitIDs) > 0 {
		available, err = s.allocations.HasAvailableUnit(ctx, snapshot.EligibleUnitIDs, pickup, returnAt)
		if err != nil {
			return Quote{}, ErrDatabaseUnavailable
		}
	}
	return Quote{
		VariantID: snapshot.VariantID, VehicleLabel: snapshot.VehicleLabel,
		PickupAt: pickup, ReturnAt: returnAt, RentalDays: days,
		DailyRateIDR: snapshot.DailyRateIDR, TotalIDR: total,
		Currency: "IDR", Available: available, QuotedAt: now.UTC(),
	}, nil
}
