package controller

import (
	"net/http"
	"os"
	"time"

	"crowdfund-app/storage"

	"github.com/labstack/echo/v4"
)

type KYCFileController struct {
	KYC        storage.KYCStore
	SignSecret []byte
}

func NewKYCFileController(kyc storage.KYCStore, secret []byte) *KYCFileController {
	return &KYCFileController{KYC: kyc, SignSecret: secret}
}

func (h *KYCFileController) Serve(c echo.Context) error {
	var (
		token string
		key   string
		f     *os.File
		ct    string
		err   error
	)
	token = c.QueryParam("token")
	key, err = storage.VerifyKYCAccess(h.SignSecret, token, time.Now().UTC())
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, apiError("invalid or expired token", "INVALID_TOKEN"))
	}
	f, ct, err = h.KYC.Open(key)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, apiError("not found", "NOT_FOUND"))
	}
	defer f.Close()
	return c.Stream(http.StatusOK, ct, f)
}
