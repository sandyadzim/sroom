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
	group := router.Group("/api/v1/booking/checkout")
	group.POST("/quotes", handler.quote)
}

func (h quoteHandler) quote(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 7*time.Second)
	defer cancel()
	requestID := c.Writer.Header().Get("X-Request-ID")

	auth := strings.Fields(c.GetHeader("Authorization"))
	if len(auth) != 2 || !strings.EqualFold(auth[0], "Bearer") {
		writeError(c, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
		return
	}
	role, err := h.identity.Role(ctx, auth[1], requestID)
	switch {
	case errors.Is(err, identity.ErrUnauthenticated):
		writeError(c, http.StatusUnauthorized, "UNAUTHENTICATED", "invalid session")
		return
	case err != nil:
		writeError(c, http.StatusServiceUnavailable, "DEPENDENCY_UNAVAILABLE", "could not verify customer session")
		return
	case role != "CUSTOMER":
		writeError(c, http.StatusForbidden, "CUSTOMER_REQUIRED", "use a customer account")
		return
	}

	payload, ok := readQuoteRequest(c)
	if !ok {
		return
	}
	pickup, pickupErr := time.Parse(time.RFC3339, payload.PickupAt)
	returnAt, returnErr := time.Parse(time.RFC3339, payload.ReturnAt)
	if pickupErr != nil || returnErr != nil {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "pickup_at and return_at must be RFC 3339 timestamps")
		return
	}
	if len(payload.VariantID) != 36 || uuid.Validate(payload.VariantID) != nil {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "variant_id must be a UUID")
		return
	}

	quote, err := h.checkout.Quote(ctx, payload.VariantID, pickup, returnAt, h.now(), requestID)
	if err != nil {
		writeQuoteError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": quote})
}

func readQuoteRequest(c *gin.Context) (quoteRequest, bool) {
	const bodyLimit = 16 * 1024
	if c.Request.ContentLength > bodyLimit {
		writeError(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "request body is too large")
		return quoteRequest{}, false
	}
	mediaType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "send an application/json request")
		return quoteRequest{}, false
	}
	body := http.MaxBytesReader(c.Writer, c.Request.Body, bodyLimit)
	defer body.Close()
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	var payload quoteRequest
	err = decoder.Decode(&payload)
	if err == nil {
		if err = decoder.Decode(new(json.RawMessage)); err == io.EOF {
			return payload, true
		}
	}
	var sizeError *http.MaxBytesError
	if errors.As(err, &sizeError) {
		writeError(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "request body is too large")
		return quoteRequest{}, false
	}
	writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "send one valid quote JSON object")
	return quoteRequest{}, false
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
