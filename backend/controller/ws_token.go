package controller

import (
	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type WSTokenController struct {
	Tokens usecase.IWSTokenUsecase
}

func NewWSTokenController(tokens usecase.IWSTokenUsecase) *WSTokenController {
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
	userID, err = requireUserID(c)
	if err != nil {
		return err
	}
	token, err = h.Tokens.Issue(reqCtx(c), userID)
	if err != nil {
		return mapUseCaseError(err)
	}
	return jsonOK(c, wsTokenResponseJSON{
		Token:     token,
		ExpiresIn: 60,
	})
}
