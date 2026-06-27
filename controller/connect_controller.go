package controller

import (
	"context"
	"net/http"

	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type ConnectUseCase interface {
	GetAccount(ctx context.Context, sellerID uuid.UUID) (*usecase.ConnectAccountView, error)
	CreateOnboardingLink(ctx context.Context, sellerID uuid.UUID) (string, error)
}

type ConnectController struct {
	Connect ConnectUseCase
}

func NewConnectController(uc ConnectUseCase) *ConnectController {
	return &ConnectController{Connect: uc}
}

func (h *ConnectController) GetAccount(c echo.Context) error {
	var (
		sellerID uuid.UUID
		row      *usecase.ConnectAccountView
		err      error
	)
	sellerID, err = CurrentUserID(c)
	if err != nil {
		return err
	}
	row, err = h.Connect.GetAccount(c.Request().Context(), sellerID)
	if err != nil {
		return mapUseCaseError(err)
	}
	if row == nil {
		return c.JSON(http.StatusOK, map[string]any{"account": nil})
	}
	return c.JSON(http.StatusOK, map[string]any{"account": toConnectAccountJSON(row)})
}

func (h *ConnectController) CreateOnboarding(c echo.Context) error {
	var (
		sellerID uuid.UUID
		url      string
		err      error
	)
	sellerID, err = CurrentUserID(c)
	if err != nil {
		return err
	}
	url, err = h.Connect.CreateOnboardingLink(c.Request().Context(), sellerID)
	if err != nil {
		return mapUseCaseError(err)
	}
	return c.JSON(http.StatusOK, map[string]string{"url": url})
}

type connectAccountResp struct {
	StripeAccountID  string `json:"stripe_account_id"`
	ChargesEnabled   bool   `json:"charges_enabled"`
	PayoutsEnabled   bool   `json:"payouts_enabled"`
	DetailsSubmitted bool   `json:"details_submitted"`
}

func toConnectAccountJSON(row *usecase.ConnectAccountView) connectAccountResp {
	if row == nil {
		return connectAccountResp{}
	}
	return connectAccountResp{
		StripeAccountID:  row.StripeAccountID,
		ChargesEnabled:   row.ChargesEnabled,
		PayoutsEnabled:   row.PayoutsEnabled,
		DetailsSubmitted: row.DetailsSubmitted,
	}
}
