package repository

import (
	"context"

	"crowdfund-app/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IPurchaseRepo interface {
	Create(ctx context.Context, row *model.Purchase) error
	Update(ctx context.Context, row *model.Purchase) error
	GetID(ctx context.Context, id uuid.UUID) (model.Purchase, error)
	GetIDLock(ctx context.Context, id uuid.UUID) (model.Purchase, error)
	GetSessionID(ctx context.Context, sessionID string) (model.Purchase, error)
	GetIntentID(ctx context.Context, intentID string) (model.Purchase, error)
	GetPendingBuyerProject(ctx context.Context, buyerID, projectID uuid.UUID) (model.Purchase, error)
	GetStripeInvoiceID(ctx context.Context, invoiceID string) (model.Purchase, error)
	ListByBuyerID(ctx context.Context, buyerID uuid.UUID, limit, offset int) ([]model.Purchase, int64, error)
	ListAll(ctx context.Context, status *model.PurchaseStatus, limit, offset int) ([]model.Purchase, int64, error)
	RecordEvent(ctx context.Context, ev *model.PurchaseEvent) error
}

type purchaseRepo struct {
	db *gorm.DB
}

func NewPurchaseRepo(db *gorm.DB) IPurchaseRepo {
	return &purchaseRepo{db: db}
}

func (r *purchaseRepo) dbCtx(ctx context.Context) *gorm.DB {
	return dbFromCtx(ctx, r.db)
}

func (r *purchaseRepo) Create(ctx context.Context, row *model.Purchase) error {
	return r.dbCtx(ctx).Create(row).Error
}

func (r *purchaseRepo) Update(ctx context.Context, row *model.Purchase) error {
	return r.dbCtx(ctx).Save(row).Error
}

func (r *purchaseRepo) GetID(ctx context.Context, id uuid.UUID) (model.Purchase, error) {
	return getOne[model.Purchase](r.dbCtx(ctx).Where("id = ?", id))
}

func (r *purchaseRepo) GetIDLock(ctx context.Context, id uuid.UUID) (model.Purchase, error) {
	return getOneForUpdate[model.Purchase](r.dbCtx(ctx).Where("id = ?", id))
}

func (r *purchaseRepo) GetSessionID(ctx context.Context, sessionID string) (model.Purchase, error) {
	return getOne[model.Purchase](r.dbCtx(ctx).Where("stripe_checkout_session_id = ?", sessionID))
}

func (r *purchaseRepo) GetIntentID(ctx context.Context, intentID string) (model.Purchase, error) {
	return getOne[model.Purchase](r.dbCtx(ctx).Where("stripe_payment_intent_id = ?", intentID))
}

func (r *purchaseRepo) GetPendingBuyerProject(ctx context.Context, buyerID, projectID uuid.UUID) (model.Purchase, error) {
	return getOne[model.Purchase](r.dbCtx(ctx).
		Where("buyer_id = ? AND project_id = ? AND status = ?", buyerID, projectID, model.PurchaseStatusPending))
}

func (r *purchaseRepo) GetStripeInvoiceID(ctx context.Context, invoiceID string) (model.Purchase, error) {
	return getOne[model.Purchase](r.dbCtx(ctx).Where("stripe_invoice_id = ?", invoiceID))
}

func (r *purchaseRepo) ListByBuyerID(ctx context.Context, buyerID uuid.UUID, limit, offset int) ([]model.Purchase, int64, error) {
	var (
		rows  []model.Purchase
		total int64
		err   error
	)
	limit, offset = paginate(limit, offset)
	err = r.dbCtx(ctx).Model(&model.Purchase{}).Where("buyer_id = ?", buyerID).Count(&total).Error
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

func (r *purchaseRepo) ListAll(ctx context.Context, status *model.PurchaseStatus, limit, offset int) ([]model.Purchase, int64, error) {
	var (
		q     *gorm.DB
		total int64
		rows  []model.Purchase
		err   error
	)
	q = r.dbCtx(ctx).Model(&model.Purchase{})
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

func (r *purchaseRepo) RecordEvent(ctx context.Context, ev *model.PurchaseEvent) error {
	return r.dbCtx(ctx).Create(ev).Error
}
