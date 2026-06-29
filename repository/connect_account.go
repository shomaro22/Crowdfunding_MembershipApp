package repository

import (
	"context"

	"crowdfund-app/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ConnectRepo interface {
	FindUserID(ctx context.Context, userID uuid.UUID) (*model.ConnectAccount, error)
	FindStripeAccountID(ctx context.Context, stripeAccountID string) (*model.ConnectAccount, error)
	Create(ctx context.Context, row *model.ConnectAccount) error
	Update(ctx context.Context, row *model.ConnectAccount) error
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

func (r *connectRepo) FindUserID(ctx context.Context, userID uuid.UUID) (*model.ConnectAccount, error) {
	return firstOnePtr[model.ConnectAccount](r.dbCtx(ctx).Where("user_id = ?", userID))
}

func (r *connectRepo) FindStripeAccountID(ctx context.Context, stripeAccountID string) (*model.ConnectAccount, error) {
	return firstOnePtr[model.ConnectAccount](r.dbCtx(ctx).Where("stripe_account_id = ?", stripeAccountID))
}

func (r *connectRepo) Create(ctx context.Context, row *model.ConnectAccount) error {
	return r.dbCtx(ctx).Create(row).Error
}

func (r *connectRepo) Update(ctx context.Context, row *model.ConnectAccount) error {
	return r.dbCtx(ctx).Save(row).Error
}
