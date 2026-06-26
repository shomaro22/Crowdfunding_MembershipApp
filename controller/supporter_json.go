package controller

import (
	"time"

	"crowdfund-app/entity"
	"crowdfund-app/usecase"
)

type upgradePlanJSON struct {
	PlanID string `json:"plan_id"`
}

type upgradeConfirmJSON struct {
	PaymentIntentID string `json:"payment_intent_id"`
}

type changeBillingJSON struct {
	BillingMode   string `json:"billing_mode"`
	PaymentMethod string `json:"payment_method"`
}

type supporterJSON struct {
	ID             string             `json:"id"`
	ProjectID      string             `json:"project_id"`
	PlanID         string             `json:"plan_id"`
	UserID         string             `json:"user_id"`
	Status         string             `json:"status"`
	BillingMode    string             `json:"billing_mode"`
	StartDate      string             `json:"start_date"`
	EndDate        string             `json:"end_date"`
	PendingPlanID  *string            `json:"pending_plan_id"`
	PlanChangeDate *string            `json:"plan_change_date"`
	Project        *projectSummaryJSON `json:"project,omitempty"`
	Plan           *projectPlanJSON   `json:"plan,omitempty"`
	PendingPlan    *projectPlanJSON   `json:"pending_plan,omitempty"`
}

type listSupportersResponse struct {
	Supporters []supporterJSON `json:"supporters"`
	Total      int             `json:"total"`
}

type upgradePlanResponse struct {
	Supporter                  supporterJSON `json:"supporter"`
	PaymentIntentClientSecret *string       `json:"payment_intent_client_secret"`
}

type purchaseJSON struct {
	ID            string              `json:"id"`
	ProjectID     string              `json:"project_id"`
	SupporterID   *string             `json:"supporter_id"`
	BuyerID       string              `json:"buyer_id"`
	Quantity      int                 `json:"quantity"`
	TotalPrice    int                 `json:"total_price"`
	Status        string              `json:"status"`
	PaymentMethod string              `json:"payment_method"`
	CreatedAt     string              `json:"created_at"`
	Project       *projectSummaryJSON `json:"project,omitempty"`
	Buyer         *userJSON           `json:"buyer,omitempty"`
}

type listPurchasesResponse struct {
	Purchases []purchaseJSON `json:"purchases"`
	Total     int64          `json:"total"`
}

func listSupportersJSON(rows []usecase.SupporterView) listSupportersResponse {
	var (
		items []supporterJSON
		i     int
	)
	items = make([]supporterJSON, 0, len(rows))
	for i = range rows {
		items = append(items, supporterToJSON(rows[i]))
	}
	return listSupportersResponse{Supporters: items, Total: len(items)}
}

func upgradePlanJSONResponse(out usecase.UpgradeOut) upgradePlanResponse {
	return upgradePlanResponse{
		Supporter:                  supporterToJSON(out.Supporter),
		PaymentIntentClientSecret: out.PaymentIntentClientSecret,
	}
}

func supporterToJSON(v usecase.SupporterView) supporterJSON {
	var (
		s      *entity.ProjectSupporter
		out    supporterJSON
		planID *string
		change *string
		proj   projectSummaryJSON
		plan   projectPlanJSON
		pend   projectPlanJSON
	)
	s = v.Supporter
	out = supporterJSON{
		ID:          s.ID.String(),
		ProjectID:   s.ProjectID.String(),
		PlanID:      s.PlanID.String(),
		UserID:      s.UserID.String(),
		Status:      string(s.Status),
		BillingMode: string(s.BillingMode),
		StartDate:   s.CurrentPeriodStart.UTC().Format(time.RFC3339),
		EndDate:     s.CurrentPeriodEnd.UTC().Format(time.RFC3339),
	}
	if s.PendingPlanID != nil {
		id := s.PendingPlanID.String()
		planID = &id
		out.PendingPlanID = planID
	}
	if s.PlanChangeDate != nil {
		t := s.PlanChangeDate.UTC().Format(time.RFC3339)
		change = &t
		out.PlanChangeDate = change
	}
	if v.Project != nil {
		proj = projectSummaryToJSON(&usecase.ProjectView{Project: v.Project, Plans: nil})
		out.Project = &proj
	}
	if v.Plan != nil {
		plan = projectPlanToJSON(v.Plan)
		out.Plan = &plan
	}
	if v.PendingPlan != nil {
		pend = projectPlanToJSON(v.PendingPlan)
		out.PendingPlan = &pend
	}
	return out
}

func projectPlanToJSON(pl *entity.ProjectPlan) projectPlanJSON {
	return projectPlanJSON{
		ID:             pl.ID.String(),
		ProjectID:      pl.ProjectID.String(),
		MembershipType: string(pl.MembershipType),
		Price:          pl.Price,
		MaxSupporters:  pl.MaxSupporters,
		Benefits:       pl.Benefits,
		IsActive:       pl.IsActive,
		SupporterCount: pl.SupporterCount,
	}
}

func listPurchasesJSON(rows []usecase.PurchaseView, total int64) listPurchasesResponse {
	var (
		items []purchaseJSON
		i     int
	)
	items = make([]purchaseJSON, 0, len(rows))
	for i = range rows {
		items = append(items, purchaseToJSON(rows[i]))
	}
	return listPurchasesResponse{Purchases: items, Total: total}
}

func purchaseToJSON(v usecase.PurchaseView) purchaseJSON {
	var (
		p      *entity.Purchase
		out    purchaseJSON
		suppID *string
		proj   projectSummaryJSON
		buyer  userJSON
	)
	p = v.Purchase
	out = purchaseJSON{
		ID:            p.ID.String(),
		ProjectID:     p.ProjectID.String(),
		BuyerID:       p.BuyerID.String(),
		Quantity:      p.Quantity,
		TotalPrice:    p.TotalPrice,
		Status:        string(p.Status),
		PaymentMethod: string(p.PaymentMethod),
		CreatedAt:     p.CreatedAt.UTC().Format(time.RFC3339),
	}
	if p.SupporterID != nil {
		id := p.SupporterID.String()
		suppID = &id
		out.SupporterID = suppID
	}
	if v.Project != nil {
		proj = projectSummaryToJSON(&usecase.ProjectView{Project: v.Project})
		out.Project = &proj
	}
	if v.Buyer != nil {
		buyer = userToJSON(v.Buyer)
		out.Buyer = &buyer
	}
	return out
}
