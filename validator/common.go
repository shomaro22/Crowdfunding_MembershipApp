package validator

import (
	"strings"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const (
	MaxOperationsRange   = 90 * 24 * time.Hour
	DefaultSecurityRange = 24 * time.Hour
	DefaultActivityRange = 30 * 24 * time.Hour
	DefaultPageLimit     = 50
	MaxPageLimit         = 100
)

func ResolveTimeRange(from, to *time.Time, defaultSpan, maxSpan time.Duration) (time.Time, time.Time, error) {
	var (
		start time.Time
		end   time.Time
		now   time.Time
		err   error
	)
	now = time.Now().UTC()
	if from == nil && to == nil {
		end = now
		start = now.Add(-defaultSpan)
		return start, end, nil
	}
	if from != nil {
		start = from.UTC()
	} else {
		start = now.Add(-defaultSpan)
	}
	if to != nil {
		end = to.UTC()
	} else {
		end = now
	}
	start, end, err = ClampTimeRange(start, end, maxSpan)
	return start, end, err
}

func ClampTimeRange(start, end time.Time, maxSpan time.Duration) (time.Time, time.Time, error) {
	if end.Before(start) {
		return time.Time{}, time.Time{}, validation.NewError("time_range", "end must not be before start")
	}
	if end.Sub(start) > maxSpan {
		return time.Time{}, time.Time{}, validation.NewError("time_range", "time range exceeds maximum allowed span")
	}
	return start, end, nil
}

func Paginate(limit, offset int) (int, int, error) {
	if offset < 0 {
		return 0, 0, validation.NewError("offset", "must be >= 0")
	}
	if limit <= 0 {
		limit = DefaultPageLimit
	}
	if limit > MaxPageLimit {
		limit = MaxPageLimit
	}
	return limit, offset, nil
}

func TrimStringPtr(s *string) *string {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil
	}
	return &v
}
