package controller

import (
	"context"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
)

type KYCFileOpener interface {
	OpenWithToken(ctx context.Context, token string) (*os.File, string, error)
}

type KYCFileController struct {
	KYC KYCFileOpener
}

func NewKYCFileController(kyc KYCFileOpener) *KYCFileController {
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
