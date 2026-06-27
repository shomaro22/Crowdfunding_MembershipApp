package usecase

import (
	"context"
	"time"

	"crowdfund-app/entity"
	"crowdfund-app/repository"

	"github.com/google/uuid"
)

func markPurchasePaidSimple(ctx context.Context, purchases repository.PurchaseRepo, purchase *entity.Purchase) error {
	var (
		from entity.PurchaseStatus
		err  error
	)
	from = purchase.Status
	purchase.Status = entity.PurchaseStatusPaid
	purchase.UpdatedAt = time.Now().UTC()
	if err = purchases.Update(ctx, purchase); err != nil {
		return err
	}
	return purchases.RecordEvent(ctx, &entity.PurchaseEvent{
		ID:         uuid.New(),
		PurchaseID: purchase.ID,
		FromStatus: &from,
		ToStatus:   entity.PurchaseStatusPaid,
		CreatedAt:  time.Now().UTC(),
	})
}
