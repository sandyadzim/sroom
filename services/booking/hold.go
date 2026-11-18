package main

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"booking/internal/checkout"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type holdRequest struct {
	quoteRequest
	ExpectedTotalIDR int64  `json:"expected_total_idr"`
	TermsVersion     string `json:"terms_version"`
	AcceptTerms      bool   `json:"accept_terms"`
}

func (h quoteHandler) createHold(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 7*time.Second)
	defer cancel()
	requestID := c.Writer.Header().Get("X-Request-ID")
	profile, ok := h.requireCustomer(c, ctx, requestID)
	if !ok {
		return
	}
	customerID, err := uuid.Parse(profile.ID)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "DEPENDENCY_UNAVAILABLE", "could not verify customer session")
		return
	}
	key := c.GetHeader("Idempotency-Key")
	if len(key) != 36 || uuid.Validate(key) != nil {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "Idempotency-Key must be a UUID")
		return
	}

	var payload holdRequest
	if !readCheckoutRequest(c, &payload) {
		return
	}
	pickup, returnAt, ok := parseQuoteSchedule(c, payload.quoteRequest)
	if !ok {
		return
	}
	if payload.ExpectedTotalIDR <= 0 {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "expected_total_idr must be positive")
		return
	}

	hold, err := h.checkout.CreateHold(ctx, checkout.HoldInput{
		CustomerID: customerID.String(), IdempotencyKey: strings.ToLower(key),
		VariantID: strings.ToLower(payload.VariantID), PickupAt: pickup, ReturnAt: returnAt,
		ExpectedTotalIDR: payload.ExpectedTotalIDR, TermsVersion: payload.TermsVersion, AcceptTerms: payload.AcceptTerms,
	}, h.now, requestID)
	if err != nil {
		writeHoldError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": hold})
}

func writeHoldError(c *gin.Context, err error) {
	var priceChanged *checkout.PriceChangedError
	switch {
	case errors.As(err, &priceChanged):
		writeErrorDetails(c, http.StatusConflict, "QUOTE_CHANGED", err.Error(), gin.H{"quote": priceChanged.Quote})
	case errors.Is(err, checkout.ErrInvalidTerms):
		writeError(c, http.StatusUnprocessableEntity, "INVALID_TERMS", err.Error())
	case errors.Is(err, checkout.ErrDemoHoldsDisabled):
		writeError(c, http.StatusServiceUnavailable, "HOLD_NOT_ENABLED", err.Error())
	case errors.Is(err, checkout.ErrAvailabilityConflict):
		writeError(c, http.StatusConflict, "AVAILABILITY_CONFLICT", err.Error())
	case errors.Is(err, checkout.ErrIdempotencyConflict):
		writeError(c, http.StatusConflict, "IDEMPOTENCY_CONFLICT", err.Error())
	case errors.Is(err, checkout.ErrHoldInProgress):
		writeError(c, http.StatusConflict, "BOOKING_IN_PROGRESS", err.Error())
	default:
		writeQuoteError(c, err)
	}
}
