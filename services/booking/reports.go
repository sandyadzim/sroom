package main

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"booking/internal/checkout"

	"github.com/gin-gonic/gin"
)

func (h quoteHandler) staffOverview(c *gin.Context) {
	h.readStaffReport(c, true)
}

func (h quoteHandler) staffReports(c *gin.Context) {
	h.readStaffReport(c, false)
}

func (h quoteHandler) readStaffReport(c *gin.Context, overview bool) {
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
	values, err := url.ParseQuery(c.Request.URL.RawQuery)
	if err != nil || !validReportQuery(values, overview) {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid report query parameters")
		return
	}
	period, err := checkout.ParseReportPeriod(values.Get("from"), values.Get("to"), h.now())
	if err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", checkout.ErrInvalidReportPeriod.Error())
		return
	}
	if overview {
		data, err := h.checkout.Overview(ctx, period)
		if err != nil {
			writeBookingReadError(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": data})
		return
	}
	data, err := h.checkout.Reports(ctx, period)
	if err != nil {
		writeBookingReadError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func validReportQuery(values url.Values, overview bool) bool {
	if overview {
		return len(values) == 0
	}
	if len(values) == 0 {
		return true
	}
	if len(values) != 2 {
		return false
	}
	for _, key := range []string{"from", "to"} {
		if len(values[key]) != 1 || values.Get(key) == "" {
			return false
		}
	}
	return true
}
