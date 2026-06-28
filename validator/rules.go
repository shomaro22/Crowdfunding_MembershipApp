package validator

import (
	"net"
	"strings"

	"crowdfund-app/entity"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

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
