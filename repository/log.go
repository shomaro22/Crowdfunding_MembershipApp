package repository

import (
	"context"
	"time"

	"crowdfund-app/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ActivityListFilter struct {
	UserID     *uuid.UUID
	Action     *entity.ActivityAction
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
	Create(ctx context.Context, row *entity.ActivityLog) error
	List(ctx context.Context, f ActivityListFilter) ([]entity.ActivityLog, int64, error)
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

func (r *activityRepo) Create(ctx context.Context, row *entity.ActivityLog) error {
	return r.dbCtx(ctx).Create(row).Error
}

func (r *activityRepo) List(ctx context.Context, f ActivityListFilter) ([]entity.ActivityLog, int64, error) {
	var (
		rows  []entity.ActivityLog
		total int64
		q     *gorm.DB
		err   error
	)
	limit, offset := paginate(f.Limit, f.Offset)
	q = r.dbCtx(ctx).Model(&entity.ActivityLog{})
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

type AuditRepo interface {
	Create(ctx context.Context, row *entity.AuditLog) error
	List(ctx context.Context, f AuditListFilter) ([]entity.AuditLog, int64, error)
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

func (r *auditRepo) Create(ctx context.Context, row *entity.AuditLog) error {
	return r.dbCtx(ctx).Create(row).Error
}

func (r *auditRepo) List(ctx context.Context, f AuditListFilter) ([]entity.AuditLog, int64, error) {
	var (
		rows  []entity.AuditLog
		total int64
		q     *gorm.DB
		err   error
	)
	limit, offset := paginate(f.Limit, f.Offset)
	q = r.dbCtx(ctx).Model(&entity.AuditLog{})
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
	Create(ctx context.Context, row *entity.ReviewDecision) error
}

type reviewDecisionRepo struct {
	db *gorm.DB
}

func NewReviewDecisionRepo(db *gorm.DB) ReviewDecisionRepo {
	return &reviewDecisionRepo{db: db}
}

func (r *reviewDecisionRepo) Create(ctx context.Context, row *entity.ReviewDecision) error {
	return dbFromCtx(ctx, r.db).Create(row).Error
}
