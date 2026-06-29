package repository

import (
	"context"
	"time"

	"crowdfund-app/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ITOTPBackupRepo interface {
	CreateBatch(ctx context.Context, rows []model.TOTPBackupCode) error
	ListUnused(ctx context.Context, userID uuid.UUID) ([]model.TOTPBackupCode, error)
	MarkUsed(ctx context.Context, id uuid.UUID, at time.Time) error
	DeleteUser(ctx context.Context, userID uuid.UUID) error
}

type totpBackupRepo struct {
	db *gorm.DB
}

func NewTOTPBackupRepo(db *gorm.DB) ITOTPBackupRepo {
	return &totpBackupRepo{db: db}
}

func (r *totpBackupRepo) CreateBatch(ctx context.Context, rows []model.TOTPBackupCode) error {
	if len(rows) == 0 {
		return nil
	}
	return dbFromCtx(ctx, r.db).Create(&rows).Error
}

func (r *totpBackupRepo) ListUnused(ctx context.Context, userID uuid.UUID) ([]model.TOTPBackupCode, error) {
	var (
		rows []model.TOTPBackupCode
		err  error
	)
	err = dbFromCtx(ctx, r.db).
		Where("user_id = ? AND used_at IS NULL", userID).
		Find(&rows).Error
	return rows, err
}

func (r *totpBackupRepo) MarkUsed(ctx context.Context, id uuid.UUID, at time.Time) error {
	return dbFromCtx(ctx, r.db).Model(&model.TOTPBackupCode{}).
		Where("id = ?", id).
		Update("used_at", at).Error
}

func (r *totpBackupRepo) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	return dbFromCtx(ctx, r.db).Where("user_id = ?", userID).Delete(&model.TOTPBackupCode{}).Error
}
