package controller

import (
	"context"
	"net/http"
	"strconv"

	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type PurchaseUseCase interface {
	ListMine(ctx context.Context, buyerID uuid.UUID, limit, offset int) ([]usecase.PurchaseView, int64, error)
}

type PurchaseController struct {
	Purchases PurchaseUseCase
}

func NewPurchaseController(uc PurchaseUseCase) *PurchaseController {
	return &PurchaseController{Purchases: uc}
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
		return mapUseCaseError(err)
	}
	return c.JSON(http.StatusOK, listPurchasesJSON(rows, total))
}
