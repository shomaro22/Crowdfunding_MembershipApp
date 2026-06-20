package repository

import (
	"context"
	"errors"

	"crowdfund-app/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SellerRegRepo interface {
	Create(ctx context.Context, r *entity.SellerReg) error
	FindByUserID(ctx context.Context, userID uuid.UUID) (*entity.SellerReg, error)
}

type sellerRegRepo struct {
	db *gorm.DB
}

func NewSellerRegRepo(db *gorm.DB) SellerRegRepo {
	return &sellerRegRepo{db: db}
}

func (r *sellerRegRepo) withDB(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok && tx != nil {
		return tx.WithContext(ctx)
	}
	return r.db.WithContext(ctx)
}

func (r *sellerRegRepo) Create(ctx context.Context, reg *entity.SellerReg) error {
	return r.withDB(ctx).Create(reg).Error
}

func (r *sellerRegRepo) FindByUserID(ctx context.Context, userID uuid.UUID) (*entity.SellerReg, error) {
	var m entity.SellerReg
	err := r.withDB(ctx).Where("user_id = ?", userID).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

type txKey struct{}

func WithTx(ctx context.Context, db *gorm.DB, fn func(ctx context.Context) error) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, txKey{}, tx))
	})
}
