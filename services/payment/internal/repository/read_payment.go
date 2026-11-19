package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"payment/internal/dto"
	"payment/internal/model"

	"gorm.io/gorm"
)

var ErrPaymentNotFound = errors.New("payment not found")

func (r *PaymentRepository) FindByBooking(ctx context.Context, bookingID string) (model.PaymentAttempt, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	var attempt model.PaymentAttempt
	err := r.db.WithContext(ctx).Table("payment.payment_attempts").Where("booking_id = ?", bookingID).Take(&attempt).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return attempt, ErrPaymentNotFound
	}
	if err != nil {
		return attempt, ErrUnavailable
	}
	return attempt, nil
}

func (r *PaymentRepository) FindByID(ctx context.Context, id string) (model.PaymentAttempt, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	var attempt model.PaymentAttempt
	err := r.db.WithContext(ctx).Table("payment.payment_attempts").Where("id = ?", id).Take(&attempt).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return attempt, ErrPaymentNotFound
	}
	if err != nil {
		return attempt, ErrUnavailable
	}
	return attempt, nil
}

func (r *PaymentRepository) List(ctx context.Context, query dto.ListPaymentsQuery) ([]model.PaymentAttempt, int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	db := r.db.WithContext(ctx).Table("payment.payment_attempts")
	if query.Search != "" {
		search := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(query.Search)
		pattern := "%" + search + "%"
		db = db.Where("(id::text ILIKE ? OR booking_id::text ILIKE ? OR provider_order_id ILIKE ?)", pattern, pattern, pattern)
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if query.Reconciliation == "required" {
		db = db.Where("review_reason IS NOT NULL")
	}
	if query.Reconciliation == "clear" {
		db = db.Where("review_reason IS NULL")
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, ErrUnavailable
	}
	attempts := make([]model.PaymentAttempt, 0)
	if err := db.Order("created_at DESC, id DESC").Offset((query.Page - 1) * query.Limit).Limit(query.Limit).Find(&attempts).Error; err != nil {
		return nil, 0, ErrUnavailable
	}
	return attempts, total, nil
}

func (r *PaymentRepository) Observations(ctx context.Context, paymentID string) ([]model.ProviderObservation, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	observations := make([]model.ProviderObservation, 0)
	// Select only fields needed by the staff timeline, never payload digests or secrets.
	err := r.db.WithContext(ctx).Table("payment.provider_observations").
		Select("id, source, provider_status, verification_result, error_code, observed_at").
		Where("payment_id = ?", paymentID).Order("observed_at ASC, id ASC").Find(&observations).Error
	if err != nil {
		return nil, ErrUnavailable
	}
	return observations, nil
}
