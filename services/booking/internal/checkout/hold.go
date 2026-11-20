package checkout

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"booking/internal/catalog"
)

var (
	ErrInvalidTerms         = errors.New("accept the current rental terms")
	ErrAvailabilityConflict = errors.New("no unit is available for this schedule")
	ErrIdempotencyConflict  = errors.New("this idempotency key was used with a different request")
	ErrHoldInProgress       = errors.New("booking request is in progress, retry with the same key")
)

type PriceChangedError struct {
	Quote Quote
}

func (e *PriceChangedError) Error() string { return "review the updated quote" }

type HoldInput struct {
	CustomerID       string
	IdempotencyKey   string
	VariantID        string
	PickupAt         time.Time
	ReturnAt         time.Time
	ExpectedTotalIDR int64
	TermsVersion     string
	AcceptTerms      bool
}

type Hold struct {
	ID            string    `json:"id"`
	Status        string    `json:"status"`
	VariantID     string    `json:"variant_id"`
	VehicleLabel  string    `json:"vehicle_label"`
	PickupAt      time.Time `json:"pickup_at"`
	ReturnAt      time.Time `json:"return_at"`
	RentalDays    int       `json:"rental_days"`
	DailyRateIDR  int64     `json:"daily_rate_idr"`
	TotalIDR      int64     `json:"total_idr"`
	Currency      string    `json:"currency"`
	HoldExpiresAt time.Time `json:"hold_expires_at"`
	PaymentStatus string    `json:"payment_status"`
}

func (s *Service) CreateHold(ctx context.Context, input HoldInput, now func() time.Time, requestID string) (Hold, error) {
	// Format the typed fields before hashing so equivalent time zones give the same fingerprint.
	fingerprint := sha256.Sum256(fmt.Appendf(nil, "%s|%s|%s|%d|%s|%t", input.VariantID,
		input.PickupAt.UTC().Format(time.RFC3339Nano), input.ReturnAt.UTC().Format(time.RFC3339Nano),
		input.ExpectedTotalIDR, input.TermsVersion, input.AcceptTerms))
	hash := hex.EncodeToString(fingerprint[:])
	key := [2]string{input.CustomerID, input.IdempotencyKey}
	if previous, exists := s.inProgress.LoadOrStore(key, hash); exists {
		if previous != hash {
			return Hold{}, ErrIdempotencyConflict
		}
		return Hold{}, ErrHoldInProgress
	}
	defer s.inProgress.Delete(key)

	saved, err := s.allocations.findHold(ctx, input, hash)
	if err != nil || saved.ID != "" {
		return s.localHold(saved), err
	}
	if !input.AcceptTerms || input.TermsVersion != "rental-terms-v1" {
		return Hold{}, ErrInvalidTerms
	}
	pickup, returnAt := input.PickupAt.In(s.wita), input.ReturnAt.In(s.wita)
	if _, err := rentalDays(pickup, returnAt, now().In(s.wita)); err != nil {
		return Hold{}, err
	}
	if s.catalog == nil {
		return Hold{}, catalog.ErrUnavailable
	}
	snapshot, err := s.catalog.BookingSnapshot(ctx, input.VariantID, requestID)
	if err != nil {
		return Hold{}, err
	}
	// Eligibility is decided by this snapshot. Later deactivation does not cancel this request.

	if err := s.allocations.rememberHoldKey(ctx, input, hash); err != nil {
		return Hold{}, err
	}
	checkedAt := now()
	days, err := rentalDays(pickup, returnAt, checkedAt.In(s.wita))
	if err != nil {
		return Hold{}, err
	}
	total, err := quoteTotal(snapshot.DailyRateIDR, days)
	if err != nil {
		return Hold{}, err
	}
	if total != input.ExpectedTotalIDR {
		quote, err := s.quoteForSnapshot(ctx, snapshot, pickup, returnAt, checkedAt, days)
		if err != nil {
			return Hold{}, err
		}
		return Hold{}, &PriceChangedError{Quote: quote}
	}
	if snapshot.Status != "active" {
		return Hold{}, ErrAvailabilityConflict
	}

	for _, unitID := range snapshot.EligibleUnitIDs {
		createdAt := now().UTC().Truncate(time.Microsecond)
		if _, err := rentalDays(pickup, returnAt, createdAt.In(s.wita)); err != nil {
			return Hold{}, err
		}
		hold := Hold{
			Status: "pending_payment", VariantID: snapshot.VariantID, VehicleLabel: snapshot.VehicleLabel,
			PickupAt: pickup, ReturnAt: returnAt, RentalDays: days, DailyRateIDR: snapshot.DailyRateIDR,
			TotalIDR: total, Currency: "IDR", HoldExpiresAt: createdAt.Add(15 * time.Minute),
		}
		saved, err := s.allocations.createHold(ctx, input, hash, hold, unitID, createdAt, requestID)
		if errors.Is(err, ErrAvailabilityConflict) {
			continue
		}
		return s.localHold(saved), err
	}
	return Hold{}, ErrAvailabilityConflict
}

func (s *Service) localHold(hold Hold) Hold {
	hold.PickupAt = hold.PickupAt.In(s.wita)
	hold.ReturnAt = hold.ReturnAt.In(s.wita)
	hold.HoldExpiresAt = hold.HoldExpiresAt.UTC()
	hold.PaymentStatus = "not_started"
	return hold
}
