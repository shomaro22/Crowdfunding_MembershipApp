package usecase

import (
	"context"
	"time"

	"crowdfund-app/model"
	"crowdfund-app/repository"

	"github.com/google/uuid"
)

func markPurchasePaidSimple(ctx context.Context, purchases repository.PurchaseRepo, purchase *model.Purchase) error {
	var (
		from model.PurchaseStatus
		err  error
	)
	from = purchase.Status
	purchase.Status = model.PurchaseStatusPaid
	purchase.UpdatedAt = time.Now().UTC()
	if err = purchases.Update(ctx, purchase); err != nil {
		return err
	}
	return purchases.RecordEvent(ctx, &model.PurchaseEvent{
		ID:         uuid.New(),
		PurchaseID: purchase.ID,
		FromStatus: &from,
		ToStatus:   model.PurchaseStatusPaid,
		CreatedAt:  time.Now().UTC(),
	})
}
