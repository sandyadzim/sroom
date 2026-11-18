package checkout

import (
	"errors"
	"math"
	"time"
	_ "time/tzdata"
)

var (
	ErrInvalidSchedule = errors.New("pickup must be tomorrow or later, return must follow pickup, and both times must be on a 30-minute slot from 08:00 to 18:00 WITA")
	ErrTotalOutOfRange = errors.New("quote total exceeds the supported IDR amount")
)

func rentalDays(pickup, returnAt, now time.Time) (int, error) {
	if !returnAt.After(pickup) || calendarDay(pickup) <= calendarDay(now) ||
		!isHandoverSlot(pickup) || !isHandoverSlot(returnAt) {
		return 0, ErrInvalidSchedule
	}
	return int(calendarDay(returnAt)-calendarDay(pickup)) + 1, nil
}

func calendarDay(local time.Time) int64 {
	year, month, day := local.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC).Unix() / 86400
}

func isHandoverSlot(local time.Time) bool {
	minute := local.Hour()*60 + local.Minute()
	return minute >= 8*60 && minute <= 18*60 && minute%30 == 0 &&
		local.Second() == 0 && local.Nanosecond() == 0
}

func quoteTotal(dailyRate int64, days int) (int64, error) {
	if dailyRate > math.MaxInt64/int64(days) {
		return 0, ErrTotalOutOfRange
	}
	return dailyRate * int64(days), nil
}
