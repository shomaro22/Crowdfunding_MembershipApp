package controller

import (
	"context"
	"net/http"
	"strconv"

	"crowdfund-app/entity"
	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type SupporterUseCase interface {
	ListMine(ctx context.Context, userID uuid.UUID) ([]usecase.SupporterView, error)
	Upgrade(ctx context.Context, userID, supporterID, planID uuid.UUID) (usecase.UpgradeOut, error)
	UpgradeConfirm(ctx context.Context, userID, supporterID uuid.UUID, intentID string) (usecase.SupporterView, error)
	Cancel(ctx context.Context, userID, supporterID uuid.UUID) (usecase.SupporterView, error)
	Rejoin(ctx context.Context, userID, supporterID uuid.UUID) (usecase.SupporterView, error)
	ChangeBilling(ctx context.Context, userID, supporterID uuid.UUID, in usecase.ChangeBillingIn) (usecase.SupporterView, error)
}

type SupporterController struct {
	Supporters SupporterUseCase
}

func NewSupporterController(uc SupporterUseCase) *SupporterController {
	return &SupporterController{Supporters: uc}
}

type PurchaseUseCase interface {
	ListMine(ctx context.Context, buyerID uuid.UUID, limit, offset int) ([]usecase.PurchaseView, int64, error)
}

type PurchaseController struct {
	Purchases PurchaseUseCase
}

func NewPurchaseController(uc PurchaseUseCase) *PurchaseController {
	return &PurchaseController{Purchases: uc}
}

func (h *SupporterController) List(c echo.Context) error {
	var (
		userID uuid.UUID
		rows   []usecase.SupporterView
		err    error
	)
	userID, err = CurrentUserID(c)
	if err != nil {
		return err
	}
	rows, err = h.Supporters.ListMine(c.Request().Context(), userID)
	if err != nil {
		return mapSupporterError(err)
	}
	return c.JSON(http.StatusOK, listSupportersJSON(rows))
}

func (h *SupporterController) Upgrade(c echo.Context) error {
	var (
		userID      uuid.UUID
		supporterID uuid.UUID
		body        upgradePlanJSON
		out         usecase.UpgradeOut
		err         error
	)
	userID, err = CurrentUserID(c)
	if err != nil {
		return err
	}
	supporterID, err = uuid.Parse(c.Param("supporterId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid supporter id", "BAD_REQUEST"))
	}
	if err = c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid request body", "BAD_REQUEST"))
	}
	planID, err := uuid.Parse(body.PlanID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid plan id", "BAD_REQUEST"))
	}
	out, err = h.Supporters.Upgrade(c.Request().Context(), userID, supporterID, planID)
	if err != nil {
		return mapSupporterError(err)
	}
	return c.JSON(http.StatusOK, upgradePlanJSONResponse(out))
}

func (h *SupporterController) UpgradeConfirm(c echo.Context) error {
	var (
		userID      uuid.UUID
		supporterID uuid.UUID
		body        upgradeConfirmJSON
		view        usecase.SupporterView
		err         error
	)
	userID, err = CurrentUserID(c)
	if err != nil {
		return err
	}
	supporterID, err = uuid.Parse(c.Param("supporterId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid supporter id", "BAD_REQUEST"))
	}
	if err = c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid request body", "BAD_REQUEST"))
	}
	view, err = h.Supporters.UpgradeConfirm(c.Request().Context(), userID, supporterID, body.PaymentIntentID)
	if err != nil {
		return mapSupporterError(err)
	}
	return c.JSON(http.StatusOK, supporterToJSON(view))
}

func (h *SupporterController) Cancel(c echo.Context) error {
	var (
		userID      uuid.UUID
		supporterID uuid.UUID
		view        usecase.SupporterView
		err         error
	)
	userID, err = CurrentUserID(c)
	if err != nil {
		return err
	}
	supporterID, err = uuid.Parse(c.Param("supporterId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid supporter id", "BAD_REQUEST"))
	}
	view, err = h.Supporters.Cancel(c.Request().Context(), userID, supporterID)
	if err != nil {
		return mapSupporterError(err)
	}
	return c.JSON(http.StatusOK, supporterToJSON(view))
}

func (h *SupporterController) Rejoin(c echo.Context) error {
	var (
		userID      uuid.UUID
		supporterID uuid.UUID
		view        usecase.SupporterView
		err         error
	)
	userID, err = CurrentUserID(c)
	if err != nil {
		return err
	}
	supporterID, err = uuid.Parse(c.Param("supporterId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid supporter id", "BAD_REQUEST"))
	}
	view, err = h.Supporters.Rejoin(c.Request().Context(), userID, supporterID)
	if err != nil {
		return mapSupporterError(err)
	}
	return c.JSON(http.StatusOK, supporterToJSON(view))
}

func (h *SupporterController) ChangeBilling(c echo.Context) error {
	var (
		userID      uuid.UUID
		supporterID uuid.UUID
		body        changeBillingJSON
		view        usecase.SupporterView
		err         error
	)
	userID, err = CurrentUserID(c)
	if err != nil {
		return err
	}
	supporterID, err = uuid.Parse(c.Param("supporterId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid supporter id", "BAD_REQUEST"))
	}
	if err = c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid request body", "BAD_REQUEST"))
	}
	view, err = h.Supporters.ChangeBilling(c.Request().Context(), userID, supporterID, usecase.ChangeBillingIn{
		BillingMode:   entity.BillingMode(body.BillingMode),
		PaymentMethod: entity.PaymentMethod(body.PaymentMethod),
	})
	if err != nil {
		return mapSupporterError(err)
	}
	return c.JSON(http.StatusOK, supporterToJSON(view))
}

func (h *PurchaseController) List(c echo.Context) error {
	var (
		userID uuid.UUID
		limit  int
		offset int
		rows   []usecase.PurchaseView
		total  int64
		err    error
	)
	userID, err = CurrentUserID(c)
	if err != nil {
		return err
	}
	limit, _ = strconv.Atoi(c.QueryParam("limit"))
	offset, _ = strconv.Atoi(c.QueryParam("offset"))
	rows, total, err = h.Purchases.ListMine(c.Request().Context(), userID, limit, offset)
	if err != nil {
		return mapSupporterError(err)
	}
	return c.JSON(http.StatusOK, listPurchasesJSON(rows, total))
}

func mapSupporterError(err error) *echo.HTTPError {
	switch err {
	case usecase.ErrForbidden:
		return echo.NewHTTPError(http.StatusForbidden, apiError("forbidden", "FORBIDDEN"))
	case usecase.ErrBadRequest:
		return echo.NewHTTPError(http.StatusBadRequest, apiError("bad request", "BAD_REQUEST"))
	case usecase.ErrNotFound:
		return echo.NewHTTPError(http.StatusNotFound, apiError("not found", "NOT_FOUND"))
	case usecase.ErrStripeUnavailable:
		return echo.NewHTTPError(http.StatusServiceUnavailable, apiError("stripe unavailable", "STRIPE_UNAVAILABLE"))
	default:
		return mapAuthError(err)
	}
}
