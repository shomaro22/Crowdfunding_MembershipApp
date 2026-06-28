package validator

import (
	"net"
	"strings"
	"time"

	"crowdfund-app/entity"

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

func activityActionRule() validation.Rule {
	return validation.By(func(value any) error {
		var (
			raw string
			a   entity.ActivityAction
		)
		raw, _ = value.(string)
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return nil
		}
		a = entity.ActivityAction(raw)
		if !a.Valid() {
			return validation.NewError("action", "invalid activity action")
		}
		return nil
	})
}

func batchStatusRule() validation.Rule {
	return validation.By(func(value any) error {
		var (
			raw string
			s   entity.BatchStatus
		)
		raw, _ = value.(string)
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return nil
		}
		s = entity.BatchStatus(raw)
		if !s.Valid() {
			return validation.NewError("status", "invalid batch status")
		}
		return nil
	})
}

func batchJobNameRule() validation.Rule {
	return validation.By(func(value any) error {
		var (
			raw string
			ok  bool
		)
		raw, _ = value.(string)
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return validation.NewError("job_name", "must not be empty")
		}
		ok = IsSupportedBatchJob(raw)
		if !ok {
			return validation.NewError("job_name", "unsupported batch job")
		}
		return nil
	})
}

func optionalBatchJobNameRule() validation.Rule {
	return validation.By(func(value any) error {
		var (
			raw string
			ok  bool
		)
		raw, _ = value.(string)
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return nil
		}
		ok = IsSupportedBatchJob(raw)
		if !ok {
			return validation.NewError("job_name", "unsupported batch job")
		}
		return nil
	})
}

func ipAddressRule() validation.Rule {
	return validation.By(func(value any) error {
		var (
			raw string
		)
		raw, _ = value.(string)
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return nil
		}
		if net.ParseIP(raw) == nil {
			return validation.NewError("ip", "invalid ip address")
		}
		return nil
	})
}

func IsSupportedBatchJob(name string) bool {
	switch strings.TrimSpace(name) {
	case entity.ApplyPlans, entity.ExpireSupporters, entity.PurgeTokens:
		return true
	default:
		return false
	}
}

func parseActivityAction(raw *string) (*entity.ActivityAction, error) {
	var (
		s string
		a entity.ActivityAction
	)
	if raw == nil {
		return nil, nil
	}
	s = strings.TrimSpace(*raw)
	if s == "" {
		return nil, nil
	}
	if err := activityActionRule().Validate(s); err != nil {
		return nil, err
	}
	a = entity.ActivityAction(s)
	return &a, nil
}

func parseBatchStatus(raw *string) (*entity.BatchStatus, error) {
	var (
		s string
		st entity.BatchStatus
	)
	if raw == nil {
		return nil, nil
	}
	s = strings.TrimSpace(*raw)
	if s == "" {
		return nil, nil
	}
	if err := batchStatusRule().Validate(s); err != nil {
		return nil, err
	}
	st = entity.BatchStatus(s)
	return &st, nil
}
