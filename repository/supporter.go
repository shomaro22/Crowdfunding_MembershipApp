package repository

import (
	"context"
	"errors"

	"crowdfund-app/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SupporterRepo interface {
	FindActiveByProjectUser(ctx context.Context, projectID, userID uuid.UUID) (*entity.ProjectSupporter, error)
	Create(ctx context.Context, row *entity.ProjectSupporter) error
	Update(ctx context.Context, row *entity.ProjectSupporter) error
}

type supporterRepo struct {
	db *gorm.DB
}

func NewSupporterRepo(db *gorm.DB) SupporterRepo {
	return &supporterRepo{db: db}
}

func (r *supporterRepo) dbCtx(ctx context.Context) *gorm.DB {
	var (
		tx *gorm.DB
		ok bool
	)
	tx, ok = ctx.Value(txKey{}).(*gorm.DB)
	if ok && tx != nil {
		return tx.WithContext(ctx)
	}
	return r.db.WithContext(ctx)
}

func (r *supporterRepo) FindActiveByProjectUser(ctx context.Context, projectID, userID uuid.UUID) (*entity.ProjectSupporter, error) {
	var (
		row entity.ProjectSupporter
		err error
	)
	err = r.dbCtx(ctx).
		Where("project_id = ? AND user_id = ? AND status = ?", projectID, userID, entity.SupporterStatusActive).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *supporterRepo) Create(ctx context.Context, row *entity.ProjectSupporter) error {
	return r.dbCtx(ctx).Create(row).Error
}

func (r *supporterRepo) Update(ctx context.Context, row *entity.ProjectSupporter) error {
	return r.dbCtx(ctx).Save(row).Error
}
