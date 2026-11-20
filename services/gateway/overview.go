package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
)

func staffOverview(client *http.Client, bookingURL, catalogURL *url.URL, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.RawQuery != "" {
			writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "overview does not accept query parameters")
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
		defer cancel()
		booking, ok := overviewPart(c, ctx, client, bookingURL, "/api/v1/booking/admin/overview", "booking", logger)
		if !ok {
			return
		}
		// Check the small composition boundary before returning a successful response.
		var data struct {
			Period *struct {
				From string `json:"from"`
				To   string `json:"to"`
			} `json:"period"`
			Timezone       string                     `json:"timezone"`
			Currency       string                     `json:"currency"`
			Simulated      *bool                      `json:"simulated"`
			Summary        map[string]json.RawMessage `json:"summary"`
			Daily          []json.RawMessage          `json:"daily"`
			RecentBookings []json.RawMessage          `json:"recent_bookings"`
		}
		raw, err := json.Marshal(booking)
		if err != nil {
			writeError(c, http.StatusBadGateway, "INVALID_UPSTREAM_RESPONSE", "invalid booking overview")
			return
		}
		if json.Unmarshal(raw, &data) != nil || data.Period == nil || data.Period.From == "" || data.Period.To == "" ||
			data.Timezone != "Asia/Makassar" || data.Currency != "IDR" || data.Simulated == nil || !*data.Simulated ||
			len(data.Summary) == 0 || data.Daily == nil || data.RecentBookings == nil {
			writeError(c, http.StatusBadGateway, "INVALID_UPSTREAM_RESPONSE", "invalid booking overview")
			return
		}
		fleet, ok := overviewPart(c, ctx, client, catalogURL, "/api/v1/catalog/admin/fleet-summary", "catalog", logger)
		if !ok {
			return
		}
		for _, key := range []string{"cars", "units"} {
			var counts struct {
				Total    *int64 `json:"total"`
				Active   *int64 `json:"active"`
				Inactive *int64 `json:"inactive"`
			}
			if json.Unmarshal(fleet[key], &counts) != nil || counts.Total == nil || counts.Active == nil || counts.Inactive == nil ||
				*counts.Total < 0 || *counts.Active < 0 || *counts.Inactive < 0 || *counts.Active > *counts.Total || *counts.Inactive != *counts.Total-*counts.Active {
				writeError(c, http.StatusBadGateway, "INVALID_UPSTREAM_RESPONSE", "invalid fleet summary")
				return
			}
		}
		fleetJSON, err := json.Marshal(fleet)
		if err != nil {
			writeError(c, http.StatusBadGateway, "INVALID_UPSTREAM_RESPONSE", "invalid fleet summary")
			return
		}
		booking["fleet"] = fleetJSON
		c.JSON(http.StatusOK, gin.H{"data": booking})
	}
}

// Forward the staff cookie as Bearer plus the request ID. Each service verifies the session.
func overviewPart(c *gin.Context, parent context.Context, client *http.Client, base *url.URL, path, service string, logger *slog.Logger) (map[string]json.RawMessage, bool) {
	ctx, cancel := context.WithTimeout(parent, 4*time.Second)
	defer cancel()
	target := *base
	target.Path = path
	target.RawQuery = ""
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		upstreamError(c, logger, service, err)
		return nil, false
	}
	request.Header.Set("Authorization", c.GetHeader("Authorization"))
	request.Header.Set("X-Request-ID", c.Writer.Header().Get("X-Request-ID"))
	response, err := client.Do(request)
	if err != nil {
		upstreamError(c, logger, service, err)
		return nil, false
	}
	defer response.Body.Close()
	const limit = 1024 * 1024
	payload, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		upstreamError(c, logger, service, err)
		return nil, false
	}
	if len(payload) > limit {
		upstreamError(c, logger, service, errors.New("overview response too large"))
		return nil, false
	}
	if response.StatusCode != http.StatusOK {
		var failure struct {
			Error *struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if response.StatusCode >= 400 && response.StatusCode <= 599 && json.Unmarshal(payload, &failure) == nil && failure.Error != nil && failure.Error.Code != "" {
			writeError(c, response.StatusCode, failure.Error.Code, failure.Error.Message)
		} else {
			writeError(c, http.StatusBadGateway, "INVALID_UPSTREAM_RESPONSE", "invalid overview response")
		}
		return nil, false
	}
	var envelope struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if json.Unmarshal(payload, &envelope) != nil || envelope.Data == nil {
		writeError(c, http.StatusBadGateway, "INVALID_UPSTREAM_RESPONSE", "invalid overview response")
		return nil, false
	}
	return envelope.Data, true
}
