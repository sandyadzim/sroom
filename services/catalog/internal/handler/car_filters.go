package handler

import (
	"math"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"catalog/internal/model"
	"catalog/internal/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func readCarFilters(c *gin.Context, admin bool) (model.CarFilter, bool) {
	query, err := url.ParseQuery(c.Request.URL.RawQuery)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid query parameters")
		return model.CarFilter{}, false
	}
	filters, ok := parseCarFilters(query, admin)
	if !ok {
		response.Error(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid car filter, sort or pagination")
	}
	return filters, ok
}

func parseCarFilters(query url.Values, admin bool) (model.CarFilter, bool) {
	filters := model.CarFilter{
		Search: strings.TrimSpace(query.Get("search")), BrandID: query.Get("brand_id"),
		Category: query.Get("category"), Transmission: query.Get("transmission"),
		Status: query.Get("status"), Sort: query.Get("sort"),
	}
	for key := range query {
		switch key {
		case "search", "brand_id", "seats", "min_price", "max_price", "category", "transmission", "sort", "page", "limit":
		case "status":
			if !admin {
				return filters, false
			}
		default:
			return filters, false
		}
	}
	if strings.ContainsRune(filters.Search, 0) ||
		(filters.BrandID != "" && (len(filters.BrandID) != 36 || uuid.Validate(filters.BrandID) != nil)) ||
		!slices.Contains([]string{"", "economy", "city", "mpv", "suv", "premium"}, filters.Category) ||
		!slices.Contains([]string{"", "automatic", "manual"}, filters.Transmission) ||
		!slices.Contains([]string{"", model.CarActive, model.CarInactive}, filters.Status) ||
		!slices.Contains([]string{"", "recommended", "price-asc", "price-desc", "name-asc"}, filters.Sort) {
		return filters, false
	}
	var ok bool
	filters.Page, filters.Limit, ok = parsePagination(query)
	if !ok {
		return filters, false
	}
	if query.Has("seats") {
		seats, err := strconv.ParseInt(query.Get("seats"), 10, 32)
		if err != nil || seats < 1 {
			return filters, false
		}
		filters.Seats = int(seats)
	}
	if query.Has("min_price") {
		price, err := strconv.ParseInt(query.Get("min_price"), 10, 64)
		if err != nil || price < 0 {
			return filters, false
		}
		filters.MinPrice = &price
	}
	if query.Has("max_price") {
		price, err := strconv.ParseInt(query.Get("max_price"), 10, 64)
		if err != nil || price < 0 {
			return filters, false
		}
		filters.MaxPrice = &price
	}
	if filters.MinPrice != nil && filters.MaxPrice != nil && *filters.MinPrice > *filters.MaxPrice {
		return filters, false
	}
	return filters, true
}

func parsePagination(query url.Values) (int, int, bool) {
	pageNumber, pageLimit := 1, 20
	if query.Has("page") {
		page, err := strconv.Atoi(query.Get("page"))
		if err != nil || page < 1 {
			return 0, 0, false
		}
		pageNumber = page
	}
	if query.Has("limit") {
		limit, err := strconv.Atoi(query.Get("limit"))
		if err != nil || limit < 1 || limit > 100 {
			return 0, 0, false
		}
		pageLimit = limit
	}
	if pageNumber-1 > math.MaxInt/pageLimit {
		return 0, 0, false
	}
	return pageNumber, pageLimit, true
}
