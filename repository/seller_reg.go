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
	Update(ctx context.Context, r *entity.SellerReg) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.SellerReg, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) (*entity.SellerReg, error)
	List(ctx context.Context, status entity.RegStatus, limit, offset int) ([]entity.SellerReg, int64, error)
}

type sellerRegRepo struct {
	db *gorm.DB
}

func NewSellerRegRepo(db *gorm.DB) SellerRegRepo {
	return &sellerRegRepo{db: db}
}

func (r *sellerRegRepo) withDB(ctx context.Context) *gorm.DB {
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

func (r *sellerRegRepo) Create(ctx context.Context, reg *entity.SellerReg) error {
	return r.withDB(ctx).Create(reg).Error
}

func (r *sellerRegRepo) Update(ctx context.Context, reg *entity.SellerReg) error {
	return r.withDB(ctx).Save(reg).Error
}

func (r *sellerRegRepo) FindByID(ctx context.Context, id uuid.UUID) (*entity.SellerReg, error) {
	var (
		m   entity.SellerReg
		err error
	)
	err = r.withDB(ctx).Where("id = ?", id).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *sellerRegRepo) FindByUserID(ctx context.Context, userID uuid.UUID) (*entity.SellerReg, error) {
	var (
		m   entity.SellerReg
		err error
	)
	err = r.withDB(ctx).Where("user_id = ?", userID).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *sellerRegRepo) List(ctx context.Context, status entity.RegStatus, limit, offset int) ([]entity.SellerReg, int64, error) {
	var (
		q     *gorm.DB
		total int64
		rows  []entity.SellerReg
		err   error
	)
	q = r.withDB(ctx).Model(&entity.SellerReg{}).Where("status = ?", status)
	err = q.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	err = r.withDB(ctx).Where("status = ?", status).
		Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error
	return rows, total, err
}

type txKey struct{}

func WithTx(ctx context.Context, db *gorm.DB, fn func(ctx context.Context) error) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, txKey{}, tx))
	})
}
