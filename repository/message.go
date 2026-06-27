package repository

import (
	"context"
	"errors"
	"time"

	"crowdfund-app/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MessageRepo interface {
	Create(ctx context.Context, row *entity.Message) error
	FindID(ctx context.Context, id uuid.UUID) (*entity.Message, error)
	ListForUser(ctx context.Context, userID uuid.UUID) ([]entity.Message, error)
	MarkRead(ctx context.Context, id uuid.UUID, at time.Time) error
	CountUnread(ctx context.Context, userID uuid.UUID) (int64, error)
}

type messageRepo struct {
	db *gorm.DB
}

func NewMessageRepo(db *gorm.DB) MessageRepo {
	return &messageRepo{db: db}
}

func (r *messageRepo) dbCtx(ctx context.Context) *gorm.DB {
	return dbFromCtx(ctx, r.db)
}

func (r *messageRepo) Create(ctx context.Context, row *entity.Message) error {
	return r.dbCtx(ctx).Create(row).Error
}

func (r *messageRepo) FindID(ctx context.Context, id uuid.UUID) (*entity.Message, error) {
	var (
		row entity.Message
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

func (r *messageRepo) ListForUser(ctx context.Context, userID uuid.UUID) ([]entity.Message, error) {
	var (
		rows []entity.Message
		err  error
	)
	err = r.dbCtx(ctx).
		Where("sender_id = ? OR receiver_id = ?", userID, userID).
		Order("created_at DESC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *messageRepo) MarkRead(ctx context.Context, id uuid.UUID, at time.Time) error {
	return r.dbCtx(ctx).Model(&entity.Message{}).
		Where("id = ? AND is_read = FALSE", id).
		Updates(map[string]any{
			"is_read": true,
			"read_at": at,
		}).Error
}

func (r *messageRepo) CountUnread(ctx context.Context, userID uuid.UUID) (int64, error) {
	var (
		n   int64
		err error
	)
	err = r.dbCtx(ctx).Model(&entity.Message{}).
		Where("receiver_id = ? AND is_read = FALSE", userID).
		Count(&n).Error
	return n, err
}
