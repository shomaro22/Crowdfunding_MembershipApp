package repository

import (
	"context"
	"errors"

	"crowdfund-app/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ConnectRepo interface {
	FindByUserID(ctx context.Context, userID uuid.UUID) (*entity.ConnectAccount, error)
	FindByStripeAccountID(ctx context.Context, stripeAccountID string) (*entity.ConnectAccount, error)
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

func (r *connectRepo) FindByUserID(ctx context.Context, userID uuid.UUID) (*entity.ConnectAccount, error) {
	var (
		row entity.ConnectAccount
		err error
	)
	err = r.dbCtx(ctx).Where("user_id = ?", userID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *connectRepo) FindByStripeAccountID(ctx context.Context, stripeAccountID string) (*entity.ConnectAccount, error) {
	var (
		row entity.ConnectAccount
		err error
	)
	err = r.dbCtx(ctx).Where("stripe_account_id = ?", stripeAccountID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *connectRepo) Create(ctx context.Context, row *entity.ConnectAccount) error {
	return r.dbCtx(ctx).Create(row).Error
}

func (r *connectRepo) Update(ctx context.Context, row *entity.ConnectAccount) error {
	return r.dbCtx(ctx).Save(row).Error
}
