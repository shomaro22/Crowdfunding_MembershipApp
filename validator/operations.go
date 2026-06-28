package validator

import (
	"strings"
	"time"

	"crowdfund-app/entity"

	"github.com/google/uuid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type ActivityLogsInput struct {
	UserID     *uuid.UUID
	Action     *string
	Resource   *string
	ResourceID *string
	IP         *string
	From       *time.Time
	To         *time.Time
	Limit      int
	Offset     int
}

type ActivityLogsNormalized struct {
	UserID     *uuid.UUID
	Action     *entity.ActivityAction
	Resource   *string
	ResourceID *string
	IP         *string
	From       time.Time
	To         time.Time
	Limit      int
	Offset     int
}

type AuditLogsInput struct {
	AdminID  *uuid.UUID
	Action   *string
	Resource *string
	Limit    int
	Offset   int
}

type AuditLogsNormalized struct {
	AdminID  *uuid.UUID
	Action   *string
	Resource *string
	Limit    int
	Offset   int
}

type BatchRunsInput struct {
	JobName *string
	Status  *string
	Limit   int
}

type BatchRunsNormalized struct {
	JobName *string
	Status  *entity.BatchStatus
	Limit   int
}

func ValidateActivityLogs(in ActivityLogsInput) (ActivityLogsNormalized, error) {
	var (
		out    ActivityLogsNormalized
		action *entity.ActivityAction
		ipRaw  string
		err    error
	)
	err = validation.ValidateStruct(&in,
		validation.Field(&in.Limit, validation.Min(0)),
		validation.Field(&in.Offset, validation.Min(0)),
	)
	if err != nil {
		return out, err
	}
	if in.Action != nil {
		action, err = parseActivityAction(in.Action)
		if err != nil {
			return out, err
		}
	}
	if in.IP != nil {
		ipRaw = strings.TrimSpace(*in.IP)
		if ipRaw != "" {
			if err = ipAddressRule().Validate(ipRaw); err != nil {
				return out, err
			}
		}
	}
	out.From, out.To, err = ResolveTimeRange(in.From, in.To, DefaultActivityRange, MaxOperationsRange)
	if err != nil {
		return out, err
	}
	out.Limit, out.Offset, err = Paginate(in.Limit, in.Offset)
	if err != nil {
		return out, err
	}
	out.UserID = in.UserID
	out.Action = action
	out.Resource = TrimStringPtr(in.Resource)
	out.ResourceID = TrimStringPtr(in.ResourceID)
	out.IP = TrimStringPtr(in.IP)
	return out, nil
}

func ValidateAuditLogs(in AuditLogsInput) (AuditLogsNormalized, error) {
	var (
		out AuditLogsNormalized
		err error
	)
	err = validation.ValidateStruct(&in,
		validation.Field(&in.Limit, validation.Min(0)),
		validation.Field(&in.Offset, validation.Min(0)),
	)
	if err != nil {
		return out, err
	}
	out.Limit, out.Offset, err = Paginate(in.Limit, in.Offset)
	if err != nil {
		return out, err
	}
	out.AdminID = in.AdminID
	out.Action = TrimStringPtr(in.Action)
	out.Resource = TrimStringPtr(in.Resource)
	return out, nil
}

func ValidateBatchRuns(in BatchRunsInput) (BatchRunsNormalized, error) {
	var (
		out    BatchRunsNormalized
		status *entity.BatchStatus
		jobRaw string
		err    error
	)
	err = validation.ValidateStruct(&in,
		validation.Field(&in.Limit, validation.Min(0)),
	)
	if err != nil {
		return out, err
	}
	if in.JobName != nil {
		jobRaw = strings.TrimSpace(*in.JobName)
		if jobRaw != "" {
			if err = optionalBatchJobNameRule().Validate(jobRaw); err != nil {
				return out, err
			}
		}
	}
	status, err = parseBatchStatus(in.Status)
	if err != nil {
		return out, err
	}
	out.Limit, _, err = Paginate(in.Limit, 0)
	if err != nil {
		return out, err
	}
	out.JobName = TrimStringPtr(in.JobName)
	out.Status = status
	return out, nil
}

func ValidateSecurityTimeRange(from, to *time.Time) (time.Time, time.Time, error) {
	return ResolveTimeRange(from, to, DefaultSecurityRange, MaxOperationsRange)
}

func ValidateBatchJobName(jobName string) (string, error) {
	var (
		raw string
		err error
	)
	raw = strings.TrimSpace(jobName)
	err = batchJobNameRule().Validate(raw)
	if err != nil {
		return "", err
	}
	return raw, nil
}
