package usecase

import (
	"context"

	"crowdfund-app/entity"
	"crowdfund-app/repository"

	"github.com/google/uuid"
)

type PurchaseUseCase struct {
	Users     repository.UserRepo
	Projects  repository.ProjectRepo
	Purchases repository.PurchaseRepo
}

func (p *PurchaseUseCase) EnrichPurchase(ctx context.Context, row *entity.Purchase) (PurchaseView, error) {
	return p.enrichPurchase(ctx, row)
}

func (p *PurchaseUseCase) ListMine(ctx context.Context, buyerID uuid.UUID, limit, offset int) ([]PurchaseView, int64, error) {
	var (
		rows  []entity.Purchase
		total int64
		out   []PurchaseView
		err   error
		i     int
	)
	rows, total, err = p.Purchases.ListByBuyerID(ctx, buyerID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	out = make([]PurchaseView, 0, len(rows))
	for i = range rows {
		var v PurchaseView
		v, err = p.enrichPurchase(ctx, &rows[i])
		if err != nil {
			return nil, 0, err
		}
		out = append(out, v)
	}
	return out, total, nil
}

func (p *PurchaseUseCase) enrichPurchase(ctx context.Context, row *entity.Purchase) (PurchaseView, error) {
	var (
		v       PurchaseView
		project *entity.Project
		buyer   *entity.User
		err     error
	)
	if row == nil {
		return PurchaseView{}, ErrNotFound
	}
	v.Purchase = PurchaseItemFrom(row)
	project, _, _, err = p.Projects.FindID(ctx, row.ProjectID)
	if err != nil {
		return PurchaseView{}, err
	}
	v.Project = ProjectItemPtrFrom(project)
	buyer, err = p.Users.FindID(ctx, row.BuyerID)
	if err != nil {
		return PurchaseView{}, err
	}
	v.Buyer = UserViewPtr(buyer)
	return v, nil
}
