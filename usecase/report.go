package usecase

import (
	"context"
	"fmt"
	"time"

	"crowdfund-app/entity"
	"crowdfund-app/repository"
	"crowdfund-app/validator"

	"github.com/google/uuid"
)

type ReportView struct {
	ID         uuid.UUID
	ReporterID uuid.UUID
	TargetType string
	TargetID   uuid.UUID
	Reason     string
	Detail     *string
	Status     string
	ReviewedBy *uuid.UUID
	ReviewedAt *time.Time
	Resolution *string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type ReportsResult struct {
	Reports []ReportView
	Total   int64
	Limit   int
	Offset  int
}

type CreateReportIn struct {
	TargetType string
	TargetID   uuid.UUID
	Reason     string
	Detail     *string
	Meta       RequestMeta
}

type UpdateReportIn struct {
	Status     string
	Resolution *string
	Reason     *string
}

type ReportUseCase struct {
	Reports  repository.ReportRepo
	Users    repository.UserRepo
	Projects repository.ProjectRepo
	Messages repository.MessageRepo
	Activity repository.ActivityRepo
	Audit    repository.AuditRepo
}

func ReportViewFrom(row entity.Report) ReportView {
	return ReportView{
		ID:         row.ID,
		ReporterID: row.ReporterID,
		TargetType: string(row.TargetType),
		TargetID:   row.TargetID,
		Reason:     row.Reason,
		Detail:     row.Detail,
		Status:     string(row.Status),
		ReviewedBy: row.ReviewedBy,
		ReviewedAt: row.ReviewedAt,
		Resolution: row.Resolution,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}
}

func (r *ReportUseCase) Create(ctx context.Context, reporterID uuid.UUID, in CreateReportIn) (ReportView, error) {
	var (
		norm   validator.CreateReportNormalized
		open   bool
		now    time.Time
		row    *entity.Report
		err    error
	)
	if r == nil || r.Reports == nil {
		return ReportView{}, ErrBadRequest
	}
	norm, err = validator.ValidateCreateReport(validator.CreateReportInput{
		TargetType: in.TargetType,
		TargetID:   in.TargetID,
		Reason:     in.Reason,
		Detail:     in.Detail,
	})
	if err != nil {
		return ReportView{}, mapReportInputErr(err)
	}
	if err = r.validateTarget(ctx, reporterID, norm.TargetType, norm.TargetID); err != nil {
		return ReportView{}, err
	}
	open, err = r.Reports.HasOpenByReporterTarget(ctx, reporterID, norm.TargetType, norm.TargetID)
	if err != nil {
		return ReportView{}, err
	}
	if open {
		return ReportView{}, ErrReportExists
	}
	now = time.Now().UTC()
	row = &entity.Report{
		ID:         uuid.New(),
		ReporterID: reporterID,
		TargetType: norm.TargetType,
		TargetID:   norm.TargetID,
		Reason:     norm.Reason,
		Detail:     norm.Detail,
		Status:     entity.ReportStatusPending,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err = r.Reports.Create(ctx, row); err != nil {
		if repository.IsReportDuplicate(err) {
			return ReportView{}, ErrReportExists
		}
		return ReportView{}, err
	}
	LogActivity(ctx, r.Activity, &reporterID, entity.ReportCreated, "report", row.ID.String(), in.Meta, ActivityMeta(map[string]string{
		"target_type": string(norm.TargetType),
		"target_id":   norm.TargetID.String(),
	}))
	return ReportViewFrom(*row), nil
}

func (r *ReportUseCase) ListAdmin(ctx context.Context, status *string, limit, offset int) (ReportsResult, error) {
	var (
		norm   validator.AdminReportListNormalized
		rows   []entity.Report
		total  int64
		out    ReportsResult
		i      int
		err    error
	)
	if r == nil || r.Reports == nil {
		return ReportsResult{}, ErrBadRequest
	}
	norm, err = validator.ValidateAdminReportList(validator.AdminReportListInput{
		Status: status,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return ReportsResult{}, mapReportInputErr(err)
	}
	rows, total, err = r.Reports.List(ctx, norm.Status, norm.Limit, norm.Offset)
	if err != nil {
		return ReportsResult{}, err
	}
	out = ReportsResult{
		Total:  total,
		Limit:  norm.Limit,
		Offset: norm.Offset,
		Reports: make([]ReportView, len(rows)),
	}
	for i = range rows {
		out.Reports[i] = ReportViewFrom(rows[i])
	}
	return out, nil
}

func (r *ReportUseCase) UpdateAdmin(ctx context.Context, adminID, reportID uuid.UUID, in UpdateReportIn) (ReportView, error) {
	var (
		row        *entity.Report
		status     entity.ReportStatus
		resolution *string
		now        time.Time
		details    string
		err        error
	)
	if r == nil || r.Reports == nil {
		return ReportView{}, ErrBadRequest
	}
	row, err = r.Reports.FindID(ctx, reportID)
	if err != nil || row == nil {
		return ReportView{}, ErrNotFound
	}
	status, resolution, err = validator.ValidateAdminReportUpdate(validator.AdminReportUpdateInput{
		Status:     in.Status,
		Resolution: in.Resolution,
	})
	if err != nil {
		return ReportView{}, mapReportInputErr(err)
	}
	now = time.Now().UTC()
	row.Status = status
	row.Resolution = resolution
	row.ReviewedBy = &adminID
	row.ReviewedAt = &now
	row.UpdatedAt = now
	if err = r.Reports.Update(ctx, row); err != nil {
		return ReportView{}, err
	}
	details = fmt.Sprintf("status=%s report_id=%s", status, reportID)
	WriteAudit(ctx, r.Audit, adminID, "report.update", "report", reportID.String(), in.Reason, &details)
	return ReportViewFrom(*row), nil
}

func (r *ReportUseCase) validateTarget(ctx context.Context, reporterID uuid.UUID, targetType entity.ReportTargetType, targetID uuid.UUID) error {
	var (
		project *entity.Project
		message *entity.Message
		user    *entity.User
		err     error
	)
	switch targetType {
	case entity.ReportTargetProject:
		project, _, _, err = r.Projects.FindID(ctx, targetID)
		if err != nil {
			return err
		}
		if project == nil {
			return ErrNotFound
		}
	case entity.ReportTargetMessage:
		if r.Messages == nil {
			return ErrBadRequest
		}
		message, err = r.Messages.FindID(ctx, targetID)
		if err != nil {
			return err
		}
		if message == nil {
			return ErrNotFound
		}
	case entity.ReportTargetUser:
		if targetID == reporterID {
			return ErrBadRequest
		}
		user, err = r.Users.FindID(ctx, targetID)
		if err != nil {
			return err
		}
		if user == nil {
			return ErrNotFound
		}
	default:
		return ErrBadRequest
	}
	return nil
}

func mapReportInputErr(err error) error {
	if err == nil {
		return nil
	}
	return ErrBadRequest
}
