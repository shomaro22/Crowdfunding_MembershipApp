package repository

import (
	"context"
	"time"

	"crowdfund-app/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ActivityListFilter struct {
	UserID     *uuid.UUID
	Action     *model.ActivityAction
	Resource   *string
	ResourceID *string
	IP         *string
	From       *time.Time
	To         *time.Time
	Limit      int
	Offset     int
}

type AuditListFilter struct {
	AdminID  *uuid.UUID
	Action   *string
	Resource *string
	Limit    int
	Offset   int
}

type ActivityRepo interface {
	Create(ctx context.Context, row *model.ActivityLog) error
	List(ctx context.Context, f ActivityListFilter) ([]model.ActivityLog, int64, error)
	CountByActionBetween(ctx context.Context, action model.ActivityAction, from, to time.Time) (int64, error)
	TopIPsBetween(ctx context.Context, from, to time.Time, limit int) ([]ActivityIPCount, error)
}

type ActivityIPCount struct {
	IP    string
	Count int64
}

type activityRepo struct {
	db *gorm.DB
}

func NewActivityRepo(db *gorm.DB) ActivityRepo {
	return &activityRepo{db: db}
}

func (r *activityRepo) dbCtx(ctx context.Context) *gorm.DB {
	return dbFromCtx(ctx, r.db)
}

func (r *activityRepo) Create(ctx context.Context, row *model.ActivityLog) error {
	return r.dbCtx(ctx).Create(row).Error
}

func (r *activityRepo) List(ctx context.Context, f ActivityListFilter) ([]model.ActivityLog, int64, error) {
	var (
		rows  []model.ActivityLog
		total int64
		q     *gorm.DB
		err   error
	)
	limit, offset := paginate(f.Limit, f.Offset)
	q = r.dbCtx(ctx).Model(&model.ActivityLog{})
	q = applyActivityListFilter(q, f)
	err = q.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	err = q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func applyActivityListFilter(q *gorm.DB, f ActivityListFilter) *gorm.DB {
	if f.UserID != nil {
		q = q.Where("user_id = ?", *f.UserID)
	}
	if f.Action != nil {
		q = q.Where("action = ?", *f.Action)
	}
	if f.Resource != nil && *f.Resource != "" {
		q = q.Where("resource = ?", *f.Resource)
	}
	if f.ResourceID != nil && *f.ResourceID != "" {
		q = q.Where("resource_id = ?", *f.ResourceID)
	}
	if f.IP != nil && *f.IP != "" {
		q = q.Where("ip_address = ?", *f.IP)
	}
	if f.From != nil {
		q = q.Where("created_at >= ?", *f.From)
	}
	if f.To != nil {
		q = q.Where("created_at <= ?", *f.To)
	}
	return q
}

func (r *activityRepo) CountByActionBetween(ctx context.Context, action model.ActivityAction, from, to time.Time) (int64, error) {
	var (
		n   int64
		err error
	)
	err = r.dbCtx(ctx).Model(&model.ActivityLog{}).
		Where("action = ? AND created_at >= ? AND created_at <= ?", action, from, to).
		Count(&n).Error
	return n, err
}

func (r *activityRepo) TopIPsBetween(ctx context.Context, from, to time.Time, limit int) ([]ActivityIPCount, error) {
	var (
		rows []ActivityIPCount
		err  error
	)
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	err = r.dbCtx(ctx).Raw(`
		SELECT ip_address AS ip, COUNT(*) AS count
		FROM activity_logs
		WHERE created_at >= ? AND created_at <= ?
		  AND ip_address IS NOT NULL
		  AND (
		    action = ?
		    OR action LIKE 'security.%'
		  )
		GROUP BY ip_address
		ORDER BY count DESC
		LIMIT ?
	`, from, to, model.UserLoginFail, limit).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

type AuditRepo interface {
	Create(ctx context.Context, row *model.AuditLog) error
	List(ctx context.Context, f AuditListFilter) ([]model.AuditLog, int64, error)
}

type auditRepo struct {
	db *gorm.DB
}

func NewAuditRepo(db *gorm.DB) AuditRepo {
	return &auditRepo{db: db}
}

func (r *auditRepo) dbCtx(ctx context.Context) *gorm.DB {
	return dbFromCtx(ctx, r.db)
}

func (r *auditRepo) Create(ctx context.Context, row *model.AuditLog) error {
	return r.dbCtx(ctx).Create(row).Error
}

func (r *auditRepo) List(ctx context.Context, f AuditListFilter) ([]model.AuditLog, int64, error) {
	var (
		rows  []model.AuditLog
		total int64
		q     *gorm.DB
		err   error
	)
	limit, offset := paginate(f.Limit, f.Offset)
	q = r.dbCtx(ctx).Model(&model.AuditLog{})
	q = applyAuditListFilter(q, f)
	err = q.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	err = q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func applyAuditListFilter(q *gorm.DB, f AuditListFilter) *gorm.DB {
	if f.AdminID != nil {
		q = q.Where("admin_id = ?", *f.AdminID)
	}
	if f.Action != nil && *f.Action != "" {
		q = q.Where("action = ?", *f.Action)
	}
	if f.Resource != nil && *f.Resource != "" {
		q = q.Where("resource = ?", *f.Resource)
	}
	return q
}

type ReviewDecisionRepo interface {
	Create(ctx context.Context, row *model.ReviewDecision) error
}

type reviewDecisionRepo struct {
	db *gorm.DB
}

func NewReviewDecisionRepo(db *gorm.DB) ReviewDecisionRepo {
	return &reviewDecisionRepo{db: db}
}

func (r *reviewDecisionRepo) Create(ctx context.Context, row *model.ReviewDecision) error {
	return dbFromCtx(ctx, r.db).Create(row).Error
}
