package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strings"
	"time"

	"booking/internal/catalog"
	"booking/internal/checkout"
	"booking/internal/identity"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type quoteHandler struct {
	identity *identity.Client
	checkout *checkout.Service
	now      func() time.Time
}

type quoteRequest struct {
	VariantID string `json:"variant_id"`
	PickupAt  string `json:"pickup_at"`
	ReturnAt  string `json:"return_at"`
}

func registerCheckout(router *gin.Engine, handler quoteHandler) {
	group := router.Group("/api/v1/booking")
	group.POST("/checkout/quotes", handler.quote)
	group.POST("/bookings", handler.createHold)
	group.GET("/bookings", handler.customerBookings)
	group.GET("/bookings/:id", handler.customerBookings)
	admin := group.Group("/admin")
	admin.GET("/bookings", handler.staffBookings)
	admin.GET("/bookings/:id", handler.staffBookings)
}

func (h quoteHandler) quote(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 7*time.Second)
	defer cancel()
	requestID := c.Writer.Header().Get("X-Request-ID")
	if _, ok := h.requireCustomer(c, ctx, requestID); !ok {
		return
	}

	var payload quoteRequest
	if !readCheckoutRequest(c, &payload) {
		return
	}
	pickup, returnAt, ok := parseQuoteSchedule(c, payload)
	if !ok {
		return
	}
	quote, err := h.checkout.Quote(ctx, payload.VariantID, pickup, returnAt, h.now(), requestID)
	if err != nil {
		writeQuoteError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": quote})
}

func (h quoteHandler) requireCustomer(c *gin.Context, ctx context.Context, requestID string) (identity.Profile, bool) {
	profile, ok := h.sessionProfile(c, ctx, requestID)
	if !ok {
		return identity.Profile{}, false
	}
	if profile.Role != "CUSTOMER" {
		writeError(c, http.StatusForbidden, "CUSTOMER_REQUIRED", "use a customer account")
		return identity.Profile{}, false
	}
	return profile, true
}

func (h quoteHandler) sessionProfile(c *gin.Context, ctx context.Context, requestID string) (identity.Profile, bool) {
	auth := strings.Fields(c.GetHeader("Authorization"))
	if len(auth) != 2 || !strings.EqualFold(auth[0], "Bearer") {
		writeError(c, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
		return identity.Profile{}, false
	}
	profile, err := h.identity.Profile(ctx, auth[1], requestID)
	switch {
	case errors.Is(err, identity.ErrUnauthenticated):
		writeError(c, http.StatusUnauthorized, "UNAUTHENTICATED", "invalid session")
	case err != nil:
		writeError(c, http.StatusServiceUnavailable, "DEPENDENCY_UNAVAILABLE", "could not verify customer session")
	default:
		return profile, true
	}
	return identity.Profile{}, false
}

func parseQuoteSchedule(c *gin.Context, payload quoteRequest) (time.Time, time.Time, bool) {
	pickup, pickupErr := time.Parse(time.RFC3339, payload.PickupAt)
	returnAt, returnErr := time.Parse(time.RFC3339, payload.ReturnAt)
	if pickupErr != nil || returnErr != nil {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "pickup_at and return_at must be RFC 3339 timestamps")
		return time.Time{}, time.Time{}, false
	}
	if len(payload.VariantID) != 36 || uuid.Validate(payload.VariantID) != nil {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "variant_id must be a UUID")
		return time.Time{}, time.Time{}, false
	}
	return pickup, returnAt, true
}

func readCheckoutRequest(c *gin.Context, payload any) bool {
	const bodyLimit = 16 * 1024
	if c.Request.ContentLength > bodyLimit {
		writeError(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "request body is too large")
		return false
	}
	mediaType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "send an application/json request")
		return false
	}
	body := http.MaxBytesReader(c.Writer, c.Request.Body, bodyLimit)
	defer body.Close()
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	err = decoder.Decode(payload)
	if err == nil {
		if err = decoder.Decode(new(json.RawMessage)); err == io.EOF {
			return true
		}
	}
	var sizeError *http.MaxBytesError
	if errors.As(err, &sizeError) {
		writeError(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "request body is too large")
		return false
	}
	writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "send one valid JSON object")
	return false
}

func writeQuoteError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, checkout.ErrInvalidSchedule):
		writeError(c, http.StatusUnprocessableEntity, "INVALID_SCHEDULE", err.Error())
	case errors.Is(err, checkout.ErrTotalOutOfRange):
		writeError(c, http.StatusUnprocessableEntity, "QUOTE_TOTAL_OUT_OF_RANGE", err.Error())
	case errors.Is(err, catalog.ErrVariantNotFound):
		writeError(c, http.StatusNotFound, "CAR_NOT_FOUND", "variant not found")
	default:
		slog.Warn("quote dependency unavailable", "service", "booking", "operation", "checkout.quote",
			"request_id", c.Writer.Header().Get("X-Request-ID"))
		writeError(c, http.StatusServiceUnavailable, "DEPENDENCY_UNAVAILABLE", "could not check rental availability")
	}
}
