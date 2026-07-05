package controller

import (
	"time"

	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type supporterController struct {
	Supporters usecase.ISupporterUsecase
}

func NewSupporterController(uc usecase.ISupporterUsecase) ISupporterController {
	return &supporterController{Supporters: uc}
}

func (h *supporterController) List(c echo.Context) error {
	var (
		userID uuid.UUID
		rows   []usecase.SupporterView
		err    error
	)
	userID, err = requireUserID(c)
	if err != nil {
		return err
	}
	rows, err = h.Supporters.ListMine(reqCtx(c), userID)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, listSupportersJSON(rows))
}

func (h *supporterController) Cancel(c echo.Context) error {
	var (
		userID      uuid.UUID
		supporterID uuid.UUID
		view        usecase.SupporterView
		err         error
	)
	userID, err = requireUserID(c)
	if err != nil {
		return err
	}
	supporterID, err = pathUUID(c, "supporterId", "invalid supporter id")
	if err != nil {
		return err
	}
	view, err = h.Supporters.Cancel(reqCtx(c), userID, supporterID)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, supporterToJSON(view))
}

func (h *supporterController) Rejoin(c echo.Context) error {
	var (
		userID      uuid.UUID
		supporterID uuid.UUID
		view        usecase.SupporterView
		err         error
	)
	userID, err = requireUserID(c)
	if err != nil {
		return err
	}
	supporterID, err = pathUUID(c, "supporterId", "invalid supporter id")
	if err != nil {
		return err
	}
	view, err = h.Supporters.Rejoin(reqCtx(c), userID, supporterID)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, supporterToJSON(view))
}

func (h *supporterController) ChangeBilling(c echo.Context) error {
	var (
		userID      uuid.UUID
		supporterID uuid.UUID
		body        changeBillingJSON
		view        usecase.SupporterView
		err         error
	)
	userID, err = requireUserID(c)
	if err != nil {
		return err
	}
	supporterID, err = pathUUID(c, "supporterId", "invalid supporter id")
	if err != nil {
		return err
	}
	if err = bindBody(c, &body); err != nil {
		return err
	}
	view, err = h.Supporters.ChangeBilling(reqCtx(c), userID, supporterID, usecase.ChangeBillingIn{
		BillingMode:   body.BillingMode,
		PaymentMethod: body.PaymentMethod,
	})
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, supporterToJSON(view))
}

type changeBillingJSON struct {
	BillingMode   string `json:"billing_mode"`
	PaymentMethod string `json:"payment_method"`
}

type supporterJSON struct {
	ID             string              `json:"id"`
	ProjectID      string              `json:"project_id"`
	PlanID         string              `json:"plan_id"`
	UserID         string              `json:"user_id"`
	Status         string              `json:"status"`
	BillingMode    string              `json:"billing_mode"`
	StartDate      string              `json:"start_date"`
	EndDate        string              `json:"end_date"`
	PendingPlanID  *string             `json:"pending_plan_id"`
	PlanChangeDate *string             `json:"plan_change_date"`
	Project        *projectSummaryJSON `json:"project,omitempty"`
	Plan           *projectPlanJSON    `json:"plan,omitempty"`
}

type listSupportersResponse struct {
	Supporters []supporterJSON `json:"supporters"`
	Total      int             `json:"total"`
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
	Buyer         *UserWire           `json:"buyer,omitempty"`
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

func supporterToJSON(v usecase.SupporterView) supporterJSON {
	var (
		s      usecase.SupporterItemView
		out    supporterJSON
		planID *string
		change *string
		project projectSummaryJSON
		plan   projectPlanJSON
	)
	s = v.Supporter
	out = supporterJSON{
		ID:          s.ID.String(),
		ProjectID:   s.ProjectID.String(),
		PlanID:      s.PlanID.String(),
		UserID:      s.UserID.String(),
		Status:      s.Status,
		BillingMode: s.BillingMode,
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
		project = projectItemToJSON(*v.Project)
		out.Project = &project
	}
	if v.Plan != nil {
		plan = projectPlanViewToJSON(*v.Plan)
		out.Plan = &plan
	}
	return out
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
		p       usecase.PurchaseItemView
		out     purchaseJSON
		suppID  *string
		project projectSummaryJSON
		buyer   UserWire
	)
	p = v.Purchase
	out = purchaseJSON{
		ID:            p.ID.String(),
		ProjectID:     p.ProjectID.String(),
		BuyerID:       p.BuyerID.String(),
		Quantity:      p.Quantity,
		TotalPrice:    p.TotalPrice,
		Status:        p.Status,
		PaymentMethod: p.PaymentMethod,
		CreatedAt:     p.CreatedAt.UTC().Format(time.RFC3339),
	}
	if p.SupporterID != nil {
		id := p.SupporterID.String()
		suppID = &id
		out.SupporterID = suppID
	}
	if v.Project != nil {
		project = projectItemToJSON(*v.Project)
		out.Project = &project
	}
	if v.Buyer != nil {
		buyer = UserWireFrom(*v.Buyer)
		out.Buyer = &buyer
	}
	return out
}
