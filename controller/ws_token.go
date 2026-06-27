package controller

import (
	"context"
	"net/http"

	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type WSTokenUseCase interface {
	Issue(ctx context.Context, userID uuid.UUID, meta usecase.RequestMeta) (string, error)
}

type WSTokenController struct {
	Tokens WSTokenUseCase
}

func NewWSTokenController(tokens WSTokenUseCase) *WSTokenController {
	return &WSTokenController{Tokens: tokens}
}

type wsTokenResponseJSON struct {
	Token     string `json:"ws_token"`
	ExpiresIn int    `json:"expires_in"`
}

func (h *WSTokenController) Issue(c echo.Context) error {
	var (
		userID uuid.UUID
		token  string
		err    error
	)
	userID, err = CurrentUserID(c)
	if err != nil {
		return err
	}
	token, err = h.Tokens.Issue(c.Request().Context(), userID, requestMeta(c))
	if err != nil {
		return mapUseCaseError(err)
	}
	return c.JSON(http.StatusOK, wsTokenResponseJSON{
		Token:     token,
		ExpiresIn: 60,
	})
}
