package repository

import (
	"context"

	"crowdfund-app/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ConnectRepo interface {
	FindUserID(ctx context.Context, userID uuid.UUID) (*entity.ConnectAccount, error)
	FindStripeAccountID(ctx context.Context, stripeAccountID string) (*entity.ConnectAccount, error)
	Create(ctx context.Context, row *entity.ConnectAccount) error
	Update(ctx context.Context, row *entity.ConnectAccount) error
}

type connectRepo struct {
	db *gorm.DB
}

func NewConnectRepo(db *gorm.DB) ConnectRepo {
	return &connectRepo{db: db}
}

func (r *connectRepo) dbCtx(ctx context.Context) *gorm.DB {
	return dbFromCtx(ctx, r.db)
}

func (r *connectRepo) FindUserID(ctx context.Context, userID uuid.UUID) (*entity.ConnectAccount, error) {
	return firstOnePtr[entity.ConnectAccount](r.dbCtx(ctx).Where("user_id = ?", userID))
}

func (r *connectRepo) FindStripeAccountID(ctx context.Context, stripeAccountID string) (*entity.ConnectAccount, error) {
	return firstOnePtr[entity.ConnectAccount](r.dbCtx(ctx).Where("stripe_account_id = ?", stripeAccountID))
}

func (r *connectRepo) Create(ctx context.Context, row *entity.ConnectAccount) error {
	return r.dbCtx(ctx).Create(row).Error
}

func (r *connectRepo) Update(ctx context.Context, row *entity.ConnectAccount) error {
	return r.dbCtx(ctx).Save(row).Error
}
