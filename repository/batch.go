package repository

import (
	"context"
	"errors"

	"crowdfund-app/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BatchRepo interface {
	Create(ctx context.Context, row *entity.BatchRun) error
	Update(ctx context.Context, row *entity.BatchRun) error
	FindID(ctx context.Context, id uuid.UUID) (*entity.BatchRun, error)
	List(ctx context.Context, jobName *string, status *entity.BatchStatus, limit int) ([]entity.BatchRun, int64, error)
}

type batchRepo struct {
	db *gorm.DB
}

func NewBatchRepo(db *gorm.DB) BatchRepo {
	return &batchRepo{db: db}
}

func (r *batchRepo) dbCtx(ctx context.Context) *gorm.DB {
	return dbFromCtx(ctx, r.db)
}

func (r *batchRepo) Create(ctx context.Context, row *entity.BatchRun) error {
	return r.dbCtx(ctx).Create(row).Error
}

func (r *batchRepo) Update(ctx context.Context, row *entity.BatchRun) error {
	return r.dbCtx(ctx).Save(row).Error
}

func (r *batchRepo) FindID(ctx context.Context, id uuid.UUID) (*entity.BatchRun, error) {
	var (
		row entity.BatchRun
		err error
	)
	err = r.dbCtx(ctx).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *batchRepo) List(ctx context.Context, jobName *string, status *entity.BatchStatus, limit int) ([]entity.BatchRun, int64, error) {
	var (
		rows  []entity.BatchRun
		total int64
		q     *gorm.DB
		err   error
	)
	limit, _ = paginate(limit, 0)
	q = r.dbCtx(ctx).Model(&entity.BatchRun{})
	if jobName != nil && *jobName != "" {
		q = q.Where("job_name = ?", *jobName)
	}
	if status != nil {
		q = q.Where("status = ?", *status)
	}
	err = q.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	err = q.Order("started_at DESC").Limit(limit).Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}
