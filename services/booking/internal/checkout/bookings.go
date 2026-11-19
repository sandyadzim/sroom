package checkout

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrBookingNotFound = errors.New("booking not found")

type Booking struct {
	ID              string    `json:"id"`
	VariantID       string    `json:"variant_id"`
	VehicleLabel    string    `json:"vehicle_label"`
	Status          string    `json:"status"`
	PickupAt        time.Time `json:"pickup_at"`
	ReturnAt        time.Time `json:"return_at"`
	RentalDays      int       `json:"rental_days"`
	DailyRateIDR    int64     `json:"daily_rate_idr" gorm:"column:daily_rate_idr"`
	TotalIDR        int64     `json:"total_idr" gorm:"column:total_idr"`
	Currency        string    `json:"currency"`
	HoldExpiresAt   time.Time `json:"hold_expires_at"`
	TermsVersion    string    `json:"terms_version"`
	TermsAcceptedAt time.Time `json:"terms_accepted_at"`
	CreatedAt       time.Time `json:"created_at"`
}

type StaffBooking struct {
	Booking    `gorm:"embedded"`
	CustomerID string  `json:"customer_id"`
	UnitID     *string `json:"unit_id"`
}

type StaffBookingDetail struct {
	StaffBooking
	StatusHistory []StatusChange `json:"status_history"`
}

type StatusChange struct {
	ID         string    `json:"id"`
	FromStatus *string   `json:"from_status"`
	ToStatus   string    `json:"to_status"`
	ActorType  string    `json:"actor_type"`
	ReasonCode *string   `json:"reason_code"`
	OccurredAt time.Time `json:"occurred_at"`
}

type BookingFilters struct {
	Status     string
	VariantID  string
	PickupDate string
	Search     string
	Page       int
	Limit      int
}

type BookingPage struct {
	Number int   `json:"number"`
	Limit  int   `json:"limit"`
	Total  int64 `json:"total"`
}

func (s *Service) ListCustomerBookings(ctx context.Context, customerID uuid.UUID, filters BookingFilters) ([]Booking, BookingPage, error) {
	rows, page, err := s.allocations.listBookings(ctx, customerID.String(), filters)
	if err != nil {
		return nil, BookingPage{}, err
	}
	bookings := make([]Booking, 0, len(rows))
	for _, row := range rows {
		bookings = append(bookings, s.localBooking(row.Booking))
	}
	return bookings, page, nil
}

func (s *Service) CustomerBooking(ctx context.Context, customerID uuid.UUID, bookingID string) (Booking, error) {
	result, err := s.allocations.findBooking(ctx, customerID.String(), bookingID)
	if err != nil {
		return Booking{}, err
	}
	return s.localBooking(result.Booking), nil
}

func (s *Service) ListStaffBookings(ctx context.Context, filters BookingFilters) ([]StaffBooking, BookingPage, error) {
	bookings, page, err := s.allocations.listBookings(ctx, "", filters)
	if err != nil {
		return nil, BookingPage{}, err
	}
	for index := range bookings {
		bookings[index].Booking = s.localBooking(bookings[index].Booking)
	}
	return bookings, page, nil
}

func (s *Service) StaffBooking(ctx context.Context, bookingID string) (StaffBookingDetail, error) {
	booking, err := s.allocations.findBooking(ctx, "", bookingID)
	if err != nil {
		return StaffBookingDetail{}, err
	}
	history, err := s.allocations.statusHistory(ctx, bookingID)
	if err != nil {
		return StaffBookingDetail{}, err
	}
	booking.Booking = s.localBooking(booking.Booking)
	for index := range history {
		history[index].OccurredAt = history[index].OccurredAt.UTC()
	}
	return StaffBookingDetail{StaffBooking: booking, StatusHistory: history}, nil
}

func (s *Service) localBooking(booking Booking) Booking {
	booking.PickupAt = booking.PickupAt.In(s.wita)
	booking.ReturnAt = booking.ReturnAt.In(s.wita)
	booking.HoldExpiresAt = booking.HoldExpiresAt.UTC()
	booking.TermsAcceptedAt = booking.TermsAcceptedAt.UTC()
	booking.CreatedAt = booking.CreatedAt.UTC()
	return booking
}
