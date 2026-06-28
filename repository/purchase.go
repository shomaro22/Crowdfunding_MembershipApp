package repository

import (
	"context"
	"errors"

	"crowdfund-app/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PurchaseRepo interface {
	Create(ctx context.Context, row *entity.Purchase) error
	Update(ctx context.Context, row *entity.Purchase) error
	FindID(ctx context.Context, id uuid.UUID) (*entity.Purchase, error)
	FindIDLock(ctx context.Context, id uuid.UUID) (*entity.Purchase, error)
	FindSessionID(ctx context.Context, sessionID string) (*entity.Purchase, error)
	FindIntentID(ctx context.Context, intentID string) (*entity.Purchase, error)
	FindPendingBuyerProject(ctx context.Context, buyerID, projectID uuid.UUID) (*entity.Purchase, error)
	FindStripeInvoiceID(ctx context.Context, invoiceID string) (*entity.Purchase, error)
	ListByBuyerID(ctx context.Context, buyerID uuid.UUID, limit, offset int) ([]entity.Purchase, int64, error)
	ListAll(ctx context.Context, status *entity.PurchaseStatus, limit, offset int) ([]entity.Purchase, int64, error)
	RecordEvent(ctx context.Context, ev *entity.PurchaseEvent) error
}

type purchaseRepo struct {
	db *gorm.DB
}

func NewPurchaseRepo(db *gorm.DB) PurchaseRepo {
	return &purchaseRepo{db: db}
}

func (r *purchaseRepo) dbCtx(ctx context.Context) *gorm.DB {
	return dbFromCtx(ctx, r.db)
}

func (r *purchaseRepo) Create(ctx context.Context, row *entity.Purchase) error {
	return r.dbCtx(ctx).Create(row).Error
}

func (r *purchaseRepo) Update(ctx context.Context, row *entity.Purchase) error {
	return r.dbCtx(ctx).Save(row).Error
}

func (r *purchaseRepo) FindID(ctx context.Context, id uuid.UUID) (*entity.Purchase, error) {
	var (
		row entity.Purchase
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

func (r *purchaseRepo) FindIDLock(ctx context.Context, id uuid.UUID) (*entity.Purchase, error) {
	var (
		row entity.Purchase
		err error
	)
	err = r.dbCtx(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *purchaseRepo) FindSessionID(ctx context.Context, sessionID string) (*entity.Purchase, error) {
	var (
		row entity.Purchase
		err error
	)
	err = r.dbCtx(ctx).Where("stripe_checkout_session_id = ?", sessionID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *purchaseRepo) FindIntentID(ctx context.Context, intentID string) (*entity.Purchase, error) {
	var (
		row entity.Purchase
		err error
	)
	err = r.dbCtx(ctx).Where("stripe_payment_intent_id = ?", intentID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *purchaseRepo) FindPendingBuyerProject(ctx context.Context, buyerID, projectID uuid.UUID) (*entity.Purchase, error) {
	var (
		row entity.Purchase
		err error
	)
	err = r.dbCtx(ctx).
		Where("buyer_id = ? AND project_id = ? AND status = ?", buyerID, projectID, entity.PurchaseStatusPending).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *purchaseRepo) FindStripeInvoiceID(ctx context.Context, invoiceID string) (*entity.Purchase, error) {
	var (
		row entity.Purchase
		err error
	)
	err = r.dbCtx(ctx).Where("stripe_invoice_id = ?", invoiceID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *purchaseRepo) ListByBuyerID(ctx context.Context, buyerID uuid.UUID, limit, offset int) ([]entity.Purchase, int64, error) {
	var (
		rows  []entity.Purchase
		total int64
		err   error
	)
	limit, offset = paginate(limit, offset)
	err = r.dbCtx(ctx).Model(&entity.Purchase{}).Where("buyer_id = ?", buyerID).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	err = r.dbCtx(ctx).Where("buyer_id = ?", buyerID).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *purchaseRepo) ListAll(ctx context.Context, status *entity.PurchaseStatus, limit, offset int) ([]entity.Purchase, int64, error) {
	var (
		q     *gorm.DB
		total int64
		rows  []entity.Purchase
		err   error
	)
	q = r.dbCtx(ctx).Model(&entity.Purchase{})
	if status != nil {
		q = q.Where("status = ?", *status)
	}
	err = q.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	limit, offset = paginate(limit, offset)
	err = q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *purchaseRepo) RecordEvent(ctx context.Context, ev *entity.PurchaseEvent) error {
	return r.dbCtx(ctx).Create(ev).Error
}
