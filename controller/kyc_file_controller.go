package controller

import (
	"net/http"
	"os"

	"crowdfund-app/usecase"

	"github.com/labstack/echo/v4"
)

type KYCFileController struct {
	KYC *usecase.KYCAccessUseCase
}

func NewKYCFileController(kyc *usecase.KYCAccessUseCase) *KYCFileController {
	return &KYCFileController{KYC: kyc}
}

func (h *KYCFileController) Serve(c echo.Context) error {
	var (
		token string
		f     *os.File
		ct    string
		err   error
	)
	token = c.QueryParam("token")
	f, ct, err = h.KYC.OpenWithToken(c.Request().Context(), token)
	if err != nil {
		if he := mapNotFound(err); he != nil {
			return he
		}
		return echo.NewHTTPError(http.StatusUnauthorized, apiError("invalid or expired token", "INVALID_TOKEN"))
	}
	defer f.Close()
	return c.Stream(http.StatusOK, ct, f)
}
