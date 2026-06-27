package repository

import (
	"context"
	"errors"
	"time"

	"crowdfund-app/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MessageThreadSummary struct {
	ProjectID   uuid.UUID `gorm:"column:project_id"`
	PartnerID   uuid.UUID `gorm:"column:partner_id"`
	LatestAt    time.Time `gorm:"column:latest_at"`
	UnreadCount int       `gorm:"column:unread_count"`
}

type MessageRepo interface {
	Create(ctx context.Context, row *entity.Message) error
	FindID(ctx context.Context, id uuid.UUID) (*entity.Message, error)
	ListForUser(ctx context.Context, userID uuid.UUID) ([]entity.Message, error)
	ListThreadSummaries(ctx context.Context, userID uuid.UUID, offset, limit int) ([]MessageThreadSummary, int64, error)
	ListMessagesInThread(ctx context.Context, userID, projectID, partnerID uuid.UUID, limit int) ([]entity.Message, error)
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

func (r *messageRepo) ListThreadSummaries(ctx context.Context, userID uuid.UUID, offset, limit int) ([]MessageThreadSummary, int64, error) {
	var (
		rows  []MessageThreadSummary
		total int64
		err   error
	)
	err = r.dbCtx(ctx).Raw(`
		SELECT COUNT(*) FROM (
			SELECT 1
			FROM messages
			WHERE sender_id = ? OR receiver_id = ?
			GROUP BY project_id, CASE WHEN sender_id = ? THEN receiver_id ELSE sender_id END
		) t
	`, userID, userID, userID).Scan(&total).Error
	if err != nil {
		return nil, 0, err
	}
	err = r.dbCtx(ctx).Raw(`
		SELECT
			project_id,
			CASE WHEN sender_id = ? THEN receiver_id ELSE sender_id END AS partner_id,
			MAX(created_at) AS latest_at,
			COUNT(*) FILTER (WHERE NOT is_read AND receiver_id = ?)::int AS unread_count
		FROM messages
		WHERE sender_id = ? OR receiver_id = ?
		GROUP BY project_id, CASE WHEN sender_id = ? THEN receiver_id ELSE sender_id END
		ORDER BY latest_at DESC
		OFFSET ? LIMIT ?
	`, userID, userID, userID, userID, userID, offset, limit).Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *messageRepo) ListMessagesInThread(ctx context.Context, userID, projectID, partnerID uuid.UUID, limit int) ([]entity.Message, error) {
	var (
		rows []entity.Message
		err  error
	)
	err = r.dbCtx(ctx).
		Where("project_id = ?", projectID).
		Where(
			"(sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)",
			userID, partnerID, partnerID, userID,
		).
		Order("created_at DESC").
		Limit(limit).
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
