package validator

import (
	"strings"
	"unicode/utf8"

	"crowdfund-app/entity"

	"github.com/google/uuid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const (
	FavoriteActionAdd    = "add"
	FavoriteActionRemove = "remove"
	MaxReportReasonLen     = 500
	MaxReportDetailLen     = 4000
	MaxReportResolutionLen = 4000
)

type FavoriteToggleInput struct {
	ProjectID uuid.UUID
	Action    string
}

func ValidateFavoriteToggle(in FavoriteToggleInput) (uuid.UUID, string, error) {
	var (
		action string
		err    error
	)
	action = strings.TrimSpace(in.Action)
	err = validation.ValidateStruct(&in,
		validation.Field(&in.ProjectID, validation.Required),
	)
	if err != nil {
		return uuid.Nil, "", err
	}
	err = validation.Validate(action, validation.Required, validation.In(FavoriteActionAdd, FavoriteActionRemove))
	if err != nil {
		return uuid.Nil, "", err
	}
	return in.ProjectID, action, nil
}

type CreateReportInput struct {
	TargetType string
	TargetID   uuid.UUID
	Reason     string
	Detail     *string
}

type CreateReportNormalized struct {
	TargetType entity.ReportTargetType
	TargetID   uuid.UUID
	Reason     string
	Detail     *string
}

type AdminReportListInput struct {
	Status *string
	Limit  int
	Offset int
}

type AdminReportListNormalized struct {
	Status *entity.ReportStatus
	Limit  int
	Offset int
}

type AdminReportUpdateInput struct {
	Status     string
	Resolution *string
}

func ValidateCreateReport(in CreateReportInput) (CreateReportNormalized, error) {
	var (
		out CreateReportNormalized
		err error
	)
	err = validation.ValidateStruct(&in,
		validation.Field(&in.TargetID, validation.Required),
		validation.Field(&in.TargetType, validation.Required),
		validation.Field(&in.Reason, validation.Required, validation.RuneLength(1, MaxReportReasonLen)),
	)
	if err != nil {
		return CreateReportNormalized{}, err
	}
	out.TargetType = entity.ReportTargetType(strings.TrimSpace(strings.ToUpper(in.TargetType)))
	if !out.TargetType.Valid() {
		return CreateReportNormalized{}, entity.ErrBadReportTarget
	}
	out.TargetID = in.TargetID
	out.Reason = strings.TrimSpace(in.Reason)
	if in.Detail != nil {
		d := strings.TrimSpace(*in.Detail)
		if d != "" {
			if runeLenReport(d) > MaxReportDetailLen {
				return CreateReportNormalized{}, entity.ErrBadReport
			}
			out.Detail = &d
		}
	}
	return out, nil
}

func ValidateAdminReportList(in AdminReportListInput) (AdminReportListNormalized, error) {
	var (
		out AdminReportListNormalized
		st  entity.ReportStatus
		err error
	)
	out.Limit, out.Offset, err = Paginate(in.Limit, in.Offset)
	if err != nil {
		return AdminReportListNormalized{}, err
	}
	if in.Status != nil && strings.TrimSpace(*in.Status) != "" {
		st = entity.ReportStatus(strings.TrimSpace(strings.ToUpper(*in.Status)))
		if !st.Valid() {
			return AdminReportListNormalized{}, entity.ErrBadReport
		}
		out.Status = &st
	}
	return out, nil
}

func ValidateAdminReportUpdate(in AdminReportUpdateInput) (entity.ReportStatus, *string, error) {
	var (
		status entity.ReportStatus
		res    *string
		err    error
	)
	err = validation.ValidateStruct(&in,
		validation.Field(&in.Status, validation.Required),
	)
	if err != nil {
		return "", nil, err
	}
	status = entity.ReportStatus(strings.TrimSpace(strings.ToUpper(in.Status)))
	if !status.Valid() {
		return "", nil, entity.ErrBadReport
	}
	if in.Resolution != nil {
		s := strings.TrimSpace(*in.Resolution)
		if s != "" {
			if runeLenReport(s) > MaxReportResolutionLen {
				return "", nil, entity.ErrBadReport
			}
			res = &s
		}
	}
	return status, res, nil
}

func runeLenReport(s string) int {
	return utf8.RuneCountInString(s)
}
