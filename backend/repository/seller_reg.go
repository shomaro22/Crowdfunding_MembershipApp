package repository

import (
	"context"

	"crowdfund-app/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ISellerRegRepo interface {
	Create(ctx context.Context, r *model.SellerReg) error
	Update(ctx context.Context, r *model.SellerReg) error
	GetID(ctx context.Context, id uuid.UUID) (model.SellerReg, error)
	GetUserID(ctx context.Context, userID uuid.UUID) (model.SellerReg, error)
	List(ctx context.Context, status model.RegStatus, limit, offset int) ([]model.SellerReg, int64, error)
}

type sellerRegRepo struct {
	db *gorm.DB
}

func NewSellerRegRepo(db *gorm.DB) ISellerRegRepo {
	return &sellerRegRepo{db: db}
}

func (r *sellerRegRepo) Create(ctx context.Context, reg *model.SellerReg) error {
	return dbFromCtx(ctx, r.db).Create(reg).Error
}

func (r *sellerRegRepo) Update(ctx context.Context, reg *model.SellerReg) error {
	return dbFromCtx(ctx, r.db).Save(reg).Error
}

func (r *sellerRegRepo) GetID(ctx context.Context, id uuid.UUID) (model.SellerReg, error) {
	return getOne[model.SellerReg](dbFromCtx(ctx, r.db).Where("id = ?", id))
}

func (r *sellerRegRepo) GetUserID(ctx context.Context, userID uuid.UUID) (model.SellerReg, error) {
	return getOne[model.SellerReg](dbFromCtx(ctx, r.db).Where("user_id = ?", userID))
}

func (r *sellerRegRepo) List(ctx context.Context, status model.RegStatus, limit, offset int) ([]model.SellerReg, int64, error) {
	var (
		q     *gorm.DB
		total int64
		rows  []model.SellerReg
		err   error
	)
	q = dbFromCtx(ctx, r.db).Model(&model.SellerReg{}).Where("status = ?", status)
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
