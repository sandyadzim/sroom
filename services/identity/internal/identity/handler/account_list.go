package handler

import (
	"errors"
	"math"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"

	"identity/internal/identity/model"
)

func accountListQuery(c fiber.Ctx) (int, int, string, string, error) {
	page, pageErr := strconv.Atoi(c.Query("page", "1"))
	limit, limitErr := strconv.Atoi(c.Query("limit", "20"))
	if pageErr != nil || limitErr != nil || page < 1 || limit < 1 || limit > 100 {
		return 0, 0, "", "", errors.New("page must be positive and limit must be between 1 and 100")
	}
	// Keep the offset calculation within the supported integer range.
	if page-1 > math.MaxInt/limit {
		return 0, 0, "", "", errors.New("page is too large")
	}

	search := strings.TrimSpace(c.Query("search"))
	status := strings.TrimSpace(c.Query("status"))
	if status != "" && status != model.StatusActive && status != model.StatusInactive {
		return 0, 0, "", "", errors.New("status must be active or inactive")
	}

	return page, limit, search, status, nil
}
