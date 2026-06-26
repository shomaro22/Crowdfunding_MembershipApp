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
	FindID(ctx context.Context, id uuid.UUID) (*entity.SellerReg, error)
	FindUserID(ctx context.Context, userID uuid.UUID) (*entity.SellerReg, error)
	List(ctx context.Context, status entity.RegStatus, limit, offset int) ([]entity.SellerReg, int64, error)
}

type sellerRegRepo struct {
	db *gorm.DB
}

func NewSellerRegRepo(db *gorm.DB) SellerRegRepo {
	return &sellerRegRepo{db: db}
}

func (r *sellerRegRepo) Create(ctx context.Context, reg *entity.SellerReg) error {
	return dbFromCtx(ctx, r.db).Create(reg).Error
}

func (r *sellerRegRepo) Update(ctx context.Context, reg *entity.SellerReg) error {
	return dbFromCtx(ctx, r.db).Save(reg).Error
}

func (r *sellerRegRepo) FindID(ctx context.Context, id uuid.UUID) (*entity.SellerReg, error) {
	var (
		m   entity.SellerReg
		err error
	)
	err = dbFromCtx(ctx, r.db).Where("id = ?", id).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *sellerRegRepo) FindUserID(ctx context.Context, userID uuid.UUID) (*entity.SellerReg, error) {
	var (
		m   entity.SellerReg
		err error
	)
	err = dbFromCtx(ctx, r.db).Where("user_id = ?", userID).First(&m).Error
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
	q = dbFromCtx(ctx, r.db).Model(&entity.SellerReg{}).Where("status = ?", status)
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
	err = dbFromCtx(ctx, r.db).Where("status = ?", status).
		Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error
	return rows, total, err
}
