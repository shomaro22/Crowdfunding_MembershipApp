package usecase

import (
	"context"

	"crowdfund-app/model"
	"crowdfund-app/repository"

	"github.com/google/uuid"
)

type PurchaseUseCase struct {
	Users     repository.IUserRepo
	Projects  repository.IProjectRepo
	Purchases repository.IPurchaseRepo
}

func (p *PurchaseUseCase) EnrichPurchase(ctx context.Context, row model.Purchase) (PurchaseView, error) {
	return p.enrichPurchase(ctx, row)
}

func (p *PurchaseUseCase) ListMine(ctx context.Context, buyerID uuid.UUID, limit, offset int) ([]PurchaseView, int64, error) {
	var (
		rows  []model.Purchase
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
		v, err = p.enrichPurchase(ctx, rows[i])
		if err != nil {
			return nil, 0, err
		}
		out = append(out, v)
	}
	return out, total, nil
}

func (p *PurchaseUseCase) enrichPurchase(ctx context.Context, row model.Purchase) (PurchaseView, error) {
	var (
		v       PurchaseView
		project model.Project
		buyer   model.User
		err     error
	)
	if row.ID == uuid.Nil {
		return PurchaseView{}, ErrNotFound
	}
	v.Purchase = PurchaseItemFrom(row)
	project, _, _, err = p.Projects.GetID(ctx, row.ProjectID)
	if err != nil {
		return PurchaseView{}, mapRepoNotFound(err)
	}
	v.Project = ProjectItemPtrFrom(project)
	buyer, err = p.Users.GetID(ctx, row.BuyerID)
	if err != nil {
		return PurchaseView{}, mapRepoNotFound(err)
	}
	v.Buyer = UserViewPtr(buyer)
	return v, nil
}
