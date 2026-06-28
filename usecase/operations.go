package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"crowdfund-app/entity"
	"crowdfund-app/repository"

	"github.com/google/uuid"
)

const (
	maxOperationsRange   = 90 * 24 * time.Hour
	defaultSecurityRange = 24 * time.Hour
	defaultActivityRange = 30 * 24 * time.Hour
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
		from, to time.Time
		action   *entity.ActivityAction
		rows     []entity.ActivityLog
		total    int64
		out      ActivityLogsResult
		limit    int
		offset   int
		i        int
		row      entity.ActivityLog
		view     ActivityLogView
		u        *entity.User
		err      error
	)
	if o == nil || o.Activity == nil {
		return ActivityLogsResult{}, ErrBadRequest
	}
	from, to, err = resolveActivityTimeRange(q.From, q.To)
	if err != nil {
		return ActivityLogsResult{}, err
	}
	if q.Action != nil {
		a := entity.ActivityAction(strings.TrimSpace(*q.Action))
		if !a.Valid() {
			return ActivityLogsResult{}, ErrBadRequest
		}
		action = &a
	}
	limit, offset = paginateQuery(q.Limit, q.Offset)
	rows, total, err = o.Activity.List(ctx, repository.ActivityListFilter{
		UserID:     q.UserID,
		Action:     action,
		Resource:   trimStringPtr(q.Resource),
		ResourceID: trimStringPtr(q.ResourceID),
		IP:         trimStringPtr(q.IP),
		From:       &from,
		To:         &to,
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		return ActivityLogsResult{}, err
	}
	out = ActivityLogsResult{
		Total:  total,
		Limit:  limit,
		Offset: offset,
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
		rows   []entity.AuditLog
		total  int64
		out    AuditLogsResult
		limit  int
		offset int
		i      int
		row    entity.AuditLog
		email  string
		u      *entity.User
		err    error
	)
	if o == nil || o.Audit == nil {
		return AuditLogsResult{}, ErrBadRequest
	}
	limit, offset = paginateQuery(q.Limit, q.Offset)
	rows, total, err = o.Audit.List(ctx, repository.AuditListFilter{
		AdminID:  q.AdminID,
		Action:   trimStringPtr(q.Action),
		Resource: trimStringPtr(q.Resource),
		Limit:    limit,
		Offset:   offset,
	})
	if err != nil {
		return AuditLogsResult{}, err
	}
	out = AuditLogsResult{
		Total:  total,
		Limit:  limit,
		Offset: offset,
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
		rows    []entity.BatchRun
		total   int64
		status  *entity.BatchStatus
		out     BatchRunsResult
		limit   int
		i       int
		view    *BatchRunView
		err     error
	)
	if o == nil || o.Runs == nil {
		return BatchRunsResult{}, ErrBadRequest
	}
	if q.Status != nil && strings.TrimSpace(*q.Status) != "" {
		s := entity.BatchStatus(strings.TrimSpace(*q.Status))
		if !s.Valid() {
			return BatchRunsResult{}, ErrBadRequest
		}
		status = &s
	}
	limit, _ = paginateQuery(q.Limit, 0)
	rows, total, err = o.Runs.List(ctx, trimStringPtr(q.JobName), status, limit)
	if err != nil {
		return BatchRunsResult{}, err
	}
	out = BatchRunsResult{Total: total, Limit: limit, Runs: make([]BatchRunView, 0, len(rows))}
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
	start, end, err = resolveSecurityTimeRange(from, to)
	if err != nil {
		return SecurityEventsView{}, err
	}
	loginFail, err = o.Activity.CountByActionBetween(ctx, entity.UserLoginFail, start, end)
	if err != nil {
		return SecurityEventsView{}, err
	}
	rateLimit, err = o.Activity.CountByActionBetween(ctx, entity.SecRateLimit, start, end)
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
		err     error
	)
	if o == nil || o.Batch == nil {
		return nil, ErrBadRequest
	}
	jobName = strings.TrimSpace(jobName)
	if jobName == "" {
		return nil, ErrBadRequest
	}
	run, err = o.Batch.Run(ctx, jobName, fmt.Sprintf("admin:%s", adminID.String()), dryRun)
	details = fmt.Sprintf("job=%s dry_run=%v", jobName, dryRun)
	if run != nil {
		details = fmt.Sprintf("%s run_id=%s status=%s", details, run.ID, run.Status)
	}
	WriteAudit(ctx, o.Audit, adminID, "batch.manual_run", "batch", jobName, nil, &details)
	return run, err
}

func resolveActivityTimeRange(from, to *time.Time) (time.Time, time.Time, error) {
	var (
		start time.Time
		end   time.Time
		now   time.Time
	)
	now = time.Now().UTC()
	if from == nil && to == nil {
		end = now
		start = now.Add(-defaultActivityRange)
		return start, end, nil
	}
	if from != nil {
		start = from.UTC()
	} else {
		start = now.Add(-defaultActivityRange)
	}
	if to != nil {
		end = to.UTC()
	} else {
		end = now
	}
	return clampTimeRange(start, end)
}

func resolveSecurityTimeRange(from, to *time.Time) (time.Time, time.Time, error) {
	var (
		start time.Time
		end   time.Time
		now   time.Time
	)
	now = time.Now().UTC()
	if from == nil && to == nil {
		end = now
		start = now.Add(-defaultSecurityRange)
		return start, end, nil
	}
	if from != nil {
		start = from.UTC()
	} else {
		start = now.Add(-defaultSecurityRange)
	}
	if to != nil {
		end = to.UTC()
	} else {
		end = now
	}
	return clampTimeRange(start, end)
}

func clampTimeRange(start, end time.Time) (time.Time, time.Time, error) {
	if end.Before(start) {
		return time.Time{}, time.Time{}, ErrBadRequest
	}
	if end.Sub(start) > maxOperationsRange {
		return time.Time{}, time.Time{}, ErrBadRequest
	}
	return start, end, nil
}

func paginateQuery(limit, offset int) (int, int) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func trimStringPtr(s *string) *string {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil
	}
	return &v
}
