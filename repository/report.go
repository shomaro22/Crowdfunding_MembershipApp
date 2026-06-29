package repository

import (
	"context"
	"errors"

	"crowdfund-app/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ReportRepo interface {
	Create(ctx context.Context, row *model.Report) error
	FindID(ctx context.Context, id uuid.UUID) (*model.Report, error)
	List(ctx context.Context, status *model.ReportStatus, limit, offset int) ([]model.Report, int64, error)
	Update(ctx context.Context, row *model.Report) error
	HasOpenByReporterTarget(ctx context.Context, reporterID uuid.UUID, targetType model.ReportTargetType, targetID uuid.UUID) (bool, error)
}

type reportRepo struct {
	db *gorm.DB
}

func NewReportRepo(db *gorm.DB) ReportRepo {
	return &reportRepo{db: db}
}

func (r *reportRepo) dbCtx(ctx context.Context) *gorm.DB {
	return dbFromCtx(ctx, r.db)
}

func (r *reportRepo) Create(ctx context.Context, row *model.Report) error {
	return r.dbCtx(ctx).Create(row).Error
}

func (r *reportRepo) FindID(ctx context.Context, id uuid.UUID) (*model.Report, error) {
	return firstOnePtr[model.Report](r.dbCtx(ctx).Where("id = ?", id))
}

func (r *reportRepo) List(ctx context.Context, status *model.ReportStatus, limit, offset int) ([]model.Report, int64, error) {
	var (
		q     *gorm.DB
		rows  []model.Report
		total int64
		err   error
	)
	q = r.dbCtx(ctx).Model(&model.Report{})
	if status != nil {
		q = q.Where("status = ?", *status)
	}
	if err = q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	err = q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *reportRepo) Update(ctx context.Context, row *model.Report) error {
	return r.dbCtx(ctx).Save(row).Error
}

func (r *reportRepo) HasOpenByReporterTarget(ctx context.Context, reporterID uuid.UUID, targetType model.ReportTargetType, targetID uuid.UUID) (bool, error) {
	var (
		n   int64
		err error
	)
	err = r.dbCtx(ctx).Model(&model.Report{}).
		Where("reporter_id = ? AND target_type = ? AND target_id = ? AND status IN ?",
			reporterID, targetType, targetID,
			[]model.ReportStatus{model.ReportStatusPending, model.ReportStatusInReview}).
		Count(&n).Error
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func IsReportDuplicate(err error) bool {
	return errors.Is(err, gorm.ErrDuplicatedKey)
}
