package controller

import (
	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type ConnectController struct {
	Connect usecase.IConnectUsecase
}

func NewConnectController(uc usecase.IConnectUsecase) *ConnectController {
	return &ConnectController{Connect: uc}
}

type connectAccountResponse struct {
	Account *connectAccountResp `json:"account"`
}

func (h *ConnectController) GetAccount(c echo.Context) error {
	var (
		sellerID uuid.UUID
		row      *usecase.ConnectAccountView
		resp     connectAccountResponse
		err      error
	)
	sellerID, err = requireUserID(c)
	if err != nil {
		return err
	}
	row, err = h.Connect.GetAccount(reqCtx(c), sellerID)
	if err != nil {
		return mapUseCaseError(err)
	}
	if row != nil {
		acct := toConnectAccountJSON(row)
		resp.Account = &acct
	}
	return jsonOK(c, resp)
}

func (h *ConnectController) CreateOnboarding(c echo.Context) error {
	var (
		sellerID uuid.UUID
		url      string
		err      error
	)
	sellerID, err = requireUserID(c)
	if err != nil {
		return err
	}
	url, err = h.Connect.CreateOnboardingLink(reqCtx(c), sellerID)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, map[string]string{"url": url})
}

type connectAccountResp struct {
	StripeAccountID  string `json:"stripe_account_id"`
	ChargesEnabled   bool   `json:"charges_enabled"`
	PayoutsEnabled   bool   `json:"payouts_enabled"`
	DetailsSubmitted bool   `json:"details_submitted"`
}

func toConnectAccountJSON(row *usecase.ConnectAccountView) connectAccountResp {
	return connectAccountResp{
		StripeAccountID:  row.StripeAccountID,
		ChargesEnabled:   row.ChargesEnabled,
		PayoutsEnabled:   row.PayoutsEnabled,
		DetailsSubmitted: row.DetailsSubmitted,
	}
}
