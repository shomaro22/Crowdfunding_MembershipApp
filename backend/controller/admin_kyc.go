package controller

import (
	"net/http"
	"os"

	"crowdfund-app/usecase"

	"github.com/labstack/echo/v4"
)

type KYCFileController struct {
	KYC usecase.IKYCAccessUsecase
}

func NewKYCFileController(kyc usecase.IKYCAccessUsecase) *KYCFileController {
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
	f, ct, err = h.KYC.OpenWithToken(reqCtx(c), token)
	if err != nil {
		return mapUseCaseError(err)
	}
	defer f.Close()
	return c.Stream(http.StatusOK, ct, f)
}
