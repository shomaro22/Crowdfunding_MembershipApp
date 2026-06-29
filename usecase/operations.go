package usecase

import (
	"context"
	"fmt"
	"time"

	"crowdfund-app/model"
	"crowdfund-app/repository"
	"crowdfund-app/validator"

	"github.com/google/uuid"
)

type ActivityLogsQuery struct {
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

type AuditLogsQuery struct {
	AdminID  *uuid.UUID
	Action   *string
	Resource *string
	Limit    int
	Offset   int
}

type BatchRunsQuery struct {
	JobName *string
	Status  *string
	Limit   int
}

type OperationsUseCase struct {
	Activity repository.ActivityRepo
	Audit    repository.AuditRepo
	Runs     repository.BatchRepo
	Users    repository.UserRepo
	Batch    *BatchUseCase
}

func (o *OperationsUseCase) ListActivityLogs(ctx context.Context, q ActivityLogsQuery) (ActivityLogsResult, error) {
	var (
		norm   validator.ActivityLogsNormalized
		rows   []model.ActivityLog
		total  int64
		out    ActivityLogsResult
		i      int
		row    model.ActivityLog
		view   ActivityLogView
		u      *model.User
		err    error
	)
	if o == nil || o.Activity == nil {
		return ActivityLogsResult{}, ErrBadRequest
	}
	norm, err = validator.ValidateActivityLogs(validator.ActivityLogsInput{
		UserID:     q.UserID,
		Action:     q.Action,
		Resource:   q.Resource,
		ResourceID: q.ResourceID,
		IP:         q.IP,
		From:       q.From,
		To:         q.To,
		Limit:      q.Limit,
		Offset:     q.Offset,
	})
	if err != nil {
		return ActivityLogsResult{}, ErrBadRequest
	}
	rows, total, err = o.Activity.List(ctx, repository.ActivityListFilter{
		UserID:     norm.UserID,
		Action:     norm.Action,
		Resource:   norm.Resource,
		ResourceID: norm.ResourceID,
		IP:         norm.IP,
		From:       &norm.From,
		To:         &norm.To,
		Limit:      norm.Limit,
		Offset:     norm.Offset,
	})
	if err != nil {
		return ActivityLogsResult{}, err
	}
	out = ActivityLogsResult{
		Total:  total,
		Limit:  norm.Limit,
		Offset: norm.Offset,
		Logs:   make([]ActivityLogView, 0, len(rows)),
	}
	for i = range rows {
		row = rows[i]
		view = ActivityLogViewFrom(row)
		if row.UserID != nil && o.Users != nil {
			u, _ = o.Users.FindID(ctx, *row.UserID)
			if u != nil {
				uv := UserViewFrom(u)
				view.User = &uv
			}
		}
		out.Logs = append(out.Logs, view)
	}
	return out, nil
}

func (o *OperationsUseCase) ListAuditLogs(ctx context.Context, q AuditLogsQuery) (AuditLogsResult, error) {
	var (
		norm   validator.AuditLogsNormalized
		rows   []model.AuditLog
		total  int64
		out    AuditLogsResult
		i      int
		row    model.AuditLog
		email  string
		u      *model.User
		err    error
	)
	if o == nil || o.Audit == nil {
		return AuditLogsResult{}, ErrBadRequest
	}
	norm, err = validator.ValidateAuditLogs(validator.AuditLogsInput{
		AdminID:  q.AdminID,
		Action:   q.Action,
		Resource: q.Resource,
		Limit:    q.Limit,
		Offset:   q.Offset,
	})
	if err != nil {
		return AuditLogsResult{}, ErrBadRequest
	}
	rows, total, err = o.Audit.List(ctx, repository.AuditListFilter{
		AdminID:  norm.AdminID,
		Action:   norm.Action,
		Resource: norm.Resource,
		Limit:    norm.Limit,
		Offset:   norm.Offset,
	})
	if err != nil {
		return AuditLogsResult{}, err
	}
	out = AuditLogsResult{
		Total:  total,
		Limit:  norm.Limit,
		Offset: norm.Offset,
		Logs:   make([]AuditLogView, 0, len(rows)),
	}
	for i = range rows {
		row = rows[i]
		email = ""
		if o.Users != nil {
			u, _ = o.Users.FindID(ctx, row.AdminID)
			if u != nil {
				email = u.Email
			}
		}
		out.Logs = append(out.Logs, AuditLogViewFrom(row, email))
	}
	return out, nil
}

func (o *OperationsUseCase) ListBatchRuns(ctx context.Context, q BatchRunsQuery) (BatchRunsResult, error) {
	var (
		norm   validator.BatchRunsNormalized
		rows   []model.BatchRun
		total  int64
		out    BatchRunsResult
		i      int
		view   *BatchRunView
		err    error
	)
	if o == nil || o.Runs == nil {
		return BatchRunsResult{}, ErrBadRequest
	}
	norm, err = validator.ValidateBatchRuns(validator.BatchRunsInput{
		JobName: q.JobName,
		Status:  q.Status,
		Limit:   q.Limit,
	})
	if err != nil {
		return BatchRunsResult{}, ErrBadRequest
	}
	rows, total, err = o.Runs.List(ctx, norm.JobName, norm.Status, norm.Limit)
	if err != nil {
		return BatchRunsResult{}, err
	}
	out = BatchRunsResult{Total: total, Limit: norm.Limit, Runs: make([]BatchRunView, 0, len(rows))}
	for i = range rows {
		view = BatchRunViewFrom(&rows[i])
		if view != nil {
			out.Runs = append(out.Runs, *view)
		}
	}
	return out, nil
}

func (o *OperationsUseCase) GetSecurityEvents(ctx context.Context, from, to *time.Time) (SecurityEventsView, error) {
	var (
		start, end time.Time
		loginFail  int64
		rateLimit  int64
		topIPs     []repository.ActivityIPCount
		out        SecurityEventsView
		i          int
		err        error
	)
	if o == nil || o.Activity == nil {
		return SecurityEventsView{}, ErrBadRequest
	}
	start, end, err = validator.ValidateSecurityTimeRange(from, to)
	if err != nil {
		return SecurityEventsView{}, ErrBadRequest
	}
	loginFail, err = o.Activity.CountByActionBetween(ctx, model.UserLoginFail, start, end)
	if err != nil {
		return SecurityEventsView{}, err
	}
	rateLimit, err = o.Activity.CountByActionBetween(ctx, model.SecRateLimit, start, end)
	if err != nil {
		return SecurityEventsView{}, err
	}
	topIPs, err = o.Activity.TopIPsBetween(ctx, start, end, 10)
	if err != nil {
		return SecurityEventsView{}, err
	}
	out = SecurityEventsView{
		LoginFailedCount: loginFail,
		RateLimitedCount: rateLimit,
		From:             start,
		To:               end,
		TopIPs:           make([]SecurityIPCountView, 0, len(topIPs)),
	}
	for i = range topIPs {
		out.TopIPs = append(out.TopIPs, SecurityIPCountView{
			IP:    topIPs[i].IP,
			Count: topIPs[i].Count,
		})
	}
	return out, nil
}

func (o *OperationsUseCase) RunBatch(ctx context.Context, adminID uuid.UUID, jobName string, dryRun bool) (*BatchRunView, error) {
	var (
		run     *BatchRunView
		details string
		valid   string
		err     error
	)
	if o == nil || o.Batch == nil {
		return nil, ErrBadRequest
	}
	valid, err = validator.ValidateBatchJobName(jobName)
	if err != nil {
		return nil, ErrBadRequest
	}
	run, err = o.Batch.Run(ctx, valid, fmt.Sprintf("admin:%s", adminID.String()), dryRun)
	details = fmt.Sprintf("job=%s dry_run=%v", valid, dryRun)
	if run != nil {
		details = fmt.Sprintf("%s run_id=%s status=%s", details, run.ID, run.Status)
	}
	WriteAudit(ctx, o.Audit, adminID, "batch.manual_run", "batch", valid, nil, &details)
	return run, err
}
