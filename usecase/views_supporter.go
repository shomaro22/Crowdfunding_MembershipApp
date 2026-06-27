package usecase

import (
	"time"

	"crowdfund-app/entity"

	"github.com/google/uuid"
)

type SupporterItemView struct {
	ID                 uuid.UUID
	ProjectID          uuid.UUID
	PlanID             uuid.UUID
	UserID             uuid.UUID
	Status             string
	BillingMode        string
	CurrentPeriodStart time.Time
	CurrentPeriodEnd   time.Time
	PendingPlanID      *uuid.UUID
	PlanChangeDate     *time.Time
}

func SupporterItemFrom(s *entity.ProjectSupporter) SupporterItemView {
	if s == nil {
		return SupporterItemView{}
	}
	return SupporterItemView{
		ID:                 s.ID,
		ProjectID:          s.ProjectID,
		PlanID:             s.PlanID,
		UserID:             s.UserID,
		Status:             string(s.Status),
		BillingMode:        string(s.BillingMode),
		CurrentPeriodStart: s.CurrentPeriodStart,
		CurrentPeriodEnd:   s.CurrentPeriodEnd,
		PendingPlanID:      s.PendingPlanID,
		PlanChangeDate:     s.PlanChangeDate,
	}
}

type SupporterView struct {
	Supporter   SupporterItemView
	Project     *ProjectItemView
	Plan        *ProjectPlanView
	PendingPlan *ProjectPlanView
}

func ProjectItemPtrFrom(p *entity.Project) *ProjectItemView {
	if p == nil {
		return nil
	}
	v := ProjectItemFrom(p)
	return &v
}

func ProjectPlanPtrFrom(plan *entity.ProjectPlan) *ProjectPlanView {
	if plan == nil {
		return nil
	}
	v := ProjectPlanFrom(*plan)
	return &v
}

type PurchaseItemView struct {
	ID            uuid.UUID
	ProjectID     uuid.UUID
	SupporterID   *uuid.UUID
	BuyerID       uuid.UUID
	Quantity      int
	TotalPrice    int
	Status        string
	PaymentMethod string
	CreatedAt     time.Time
}

func PurchaseItemFrom(p *entity.Purchase) PurchaseItemView {
	if p == nil {
		return PurchaseItemView{}
	}
	return PurchaseItemView{
		ID:            p.ID,
		ProjectID:     p.ProjectID,
		SupporterID:   p.SupporterID,
		BuyerID:       p.BuyerID,
		Quantity:      p.Quantity,
		TotalPrice:    p.TotalPrice,
		Status:        string(p.Status),
		PaymentMethod: string(p.PaymentMethod),
		CreatedAt:     p.CreatedAt,
	}
}

type PurchaseView struct {
	Purchase PurchaseItemView
	Project  *ProjectItemView
	Buyer    *UserView
}
