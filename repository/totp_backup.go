package repository

import (
	"context"
	"time"

	"crowdfund-app/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TOTPBackupRepo interface {
	CreateBatch(ctx context.Context, rows []entity.TOTPBackupCode) error
	ListUnused(ctx context.Context, userID uuid.UUID) ([]entity.TOTPBackupCode, error)
	MarkUsed(ctx context.Context, id uuid.UUID, at time.Time) error
	DeleteUser(ctx context.Context, userID uuid.UUID) error
}

type totpBackupRepo struct {
	db *gorm.DB
}

func NewTOTPBackupRepo(db *gorm.DB) TOTPBackupRepo {
	return &totpBackupRepo{db: db}
}

func (r *totpBackupRepo) CreateBatch(ctx context.Context, rows []entity.TOTPBackupCode) error {
	if len(rows) == 0 {
		return nil
	}
	return dbFromCtx(ctx, r.db).Create(&rows).Error
}

func (r *totpBackupRepo) ListUnused(ctx context.Context, userID uuid.UUID) ([]entity.TOTPBackupCode, error) {
	var (
		rows []entity.TOTPBackupCode
		err  error
	)
	err = dbFromCtx(ctx, r.db).
		Where("user_id = ? AND used_at IS NULL", userID).
		Find(&rows).Error
	return rows, err
}

func (r *totpBackupRepo) MarkUsed(ctx context.Context, id uuid.UUID, at time.Time) error {
	return dbFromCtx(ctx, r.db).Model(&entity.TOTPBackupCode{}).
		Where("id = ?", id).
		Update("used_at", at).Error
}

func (r *totpBackupRepo) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	return dbFromCtx(ctx, r.db).Where("user_id = ?", userID).Delete(&entity.TOTPBackupCode{}).Error
}
