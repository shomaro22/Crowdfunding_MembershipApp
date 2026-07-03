package usecase

import (
	"context"
	"time"

	"crowdfund-app/model"
	"crowdfund-app/repository"

	"github.com/google/uuid"
)

// MarkPaidOptions は購入を PAID に遷移させる際のオプション。
type MarkPaidOptions struct {
	SupporterID *uuid.UUID
	PeriodStart *time.Time
	PeriodEnd   *time.Time
	InvoiceID   *string
	AddFunding  bool
}

// MarkPurchasePaid は購入ステータスを PAID にし、必要ならプロジェクト集計を更新する。
func MarkPurchasePaid(
	ctx context.Context,
	purchases repository.IPurchaseRepo,
	projects repository.IProjectRepo,
	purchase model.Purchase,
	opts MarkPaidOptions,
) error {
	var (
		from model.PurchaseStatus
		err  error
	)
	from = purchase.Status
	purchase.Status = model.PurchaseStatusPaid
	if opts.SupporterID != nil {
		purchase.SupporterID = opts.SupporterID
	}
	if opts.PeriodStart != nil {
		purchase.PeriodStart = opts.PeriodStart
	}
	if opts.PeriodEnd != nil {
		purchase.PeriodEnd = opts.PeriodEnd
	}
	if opts.InvoiceID != nil {
		purchase.StripeInvoiceID = opts.InvoiceID
	}
	purchase.UpdatedAt = time.Now().UTC()
	if err = purchases.Update(ctx, &purchase); err != nil {
		return err
	}
	if opts.AddFunding && projects != nil {
		if err = projects.AddCurrentAmount(ctx, purchase.ProjectID, int64(purchase.TotalPrice)); err != nil {
			return err
		}
	}
	return purchases.RecordEvent(ctx, &model.PurchaseEvent{
		ID:         uuid.New(),
		PurchaseID: purchase.ID,
		FromStatus: &from,
		ToStatus:   model.PurchaseStatusPaid,
		CreatedAt:  time.Now().UTC(),
	})
}
