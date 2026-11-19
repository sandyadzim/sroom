package main

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"booking/internal/checkout"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h quoteHandler) customerBookings(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 7*time.Second)
	defer cancel()
	profile, ok := h.requireCustomer(c, ctx, c.Writer.Header().Get("X-Request-ID"))
	if !ok {
		return
	}
	customerID, err := uuid.Parse(profile.ID)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "DEPENDENCY_UNAVAILABLE", "could not verify customer session")
		return
	}
	if id := c.Param("id"); id != "" {
		if !validBookingID(id) {
			writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "booking id must be a UUID")
			return
		}
		booking, err := h.checkout.CustomerBooking(ctx, customerID, id)
		if err != nil {
			writeBookingReadError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": booking})
		return
	}
	filters, ok := readBookingFilters(c, false)
	if !ok {
		return
	}
	bookings, page, err := h.checkout.ListCustomerBookings(ctx, customerID, filters)
	if err != nil {
		writeBookingReadError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": bookings, "page": page})
}

func (h quoteHandler) staffBookings(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 7*time.Second)
	defer cancel()
	profile, ok := h.sessionProfile(c, ctx, c.Writer.Header().Get("X-Request-ID"))
	if !ok {
		return
	}
	if profile.Role != "ADMIN" && profile.Role != "OWNER" {
		writeError(c, http.StatusForbidden, "FORBIDDEN", "staff access required")
		return
	}
	if id := c.Param("id"); id != "" {
		if !validBookingID(id) {
			writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "booking id must be a UUID")
			return
		}
		booking, err := h.checkout.StaffBooking(ctx, id)
		if err != nil {
			writeBookingReadError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": booking})
		return
	}
	filters, ok := readBookingFilters(c, true)
	if !ok {
		return
	}
	bookings, page, err := h.checkout.ListStaffBookings(ctx, filters)
	if err != nil {
		writeBookingReadError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": bookings, "page": page})
}

func readBookingFilters(c *gin.Context, staff bool) (checkout.BookingFilters, bool) {
	values, err := url.ParseQuery(c.Request.URL.RawQuery)
	if err == nil {
		filters, err := parseBookingFilters(values, staff)
		if err == nil {
			return filters, true
		}
	}
	writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid booking filters or pagination")
	return checkout.BookingFilters{}, false
}

func parseBookingFilters(values url.Values, staff bool) (checkout.BookingFilters, error) {
	invalid := errors.New("invalid booking filters")
	filters := checkout.BookingFilters{Page: 1, Limit: 20}
	for name, items := range values {
		if len(items) != 1 {
			return filters, invalid
		}
		value := items[0]
		switch name {
		case "page":
			page, err := strconv.Atoi(value)
			if err != nil || page < 1 {
				return filters, invalid
			}
			filters.Page = page
		case "limit":
			limit, err := strconv.Atoi(value)
			if err != nil || limit < 1 || limit > 100 {
				return filters, invalid
			}
			filters.Limit = limit
		case "status":
			switch value {
			case "pending_payment", "confirmed", "expired", "payment_review":
				filters.Status = value
			default:
				return filters, invalid
			}
		case "variant_id":
			if !staff || !validBookingID(value) {
				return filters, invalid
			}
			filters.VariantID = value
		case "pickup_date":
			date, err := time.Parse(time.DateOnly, value)
			if !staff || err != nil || date.Year() < 1 {
				return filters, invalid
			}
			filters.PickupDate = value
		case "search":
			if !staff {
				return filters, invalid
			}
			filters.Search = strings.TrimSpace(value)
		default:
			return filters, invalid
		}
	}
	if filters.Page-1 > math.MaxInt/filters.Limit {
		return filters, invalid
	}
	return filters, nil
}

func validBookingID(value string) bool {
	return len(value) == 36 && uuid.Validate(value) == nil
}

func writeBookingReadError(c *gin.Context, err error) {
	if errors.Is(err, checkout.ErrBookingNotFound) {
		writeError(c, http.StatusNotFound, "BOOKING_NOT_FOUND", err.Error())
		return
	}
	slog.Warn("booking read unavailable", "service", "booking", "operation", "bookings.read",
		"request_id", c.Writer.Header().Get("X-Request-ID"))
	writeError(c, http.StatusServiceUnavailable, "DEPENDENCY_UNAVAILABLE", "could not read bookings")
}
