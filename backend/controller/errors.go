package controller

import (
	"crowdfund-app/httperr"

	"github.com/labstack/echo/v4"
)

func mapUseCaseError(err error) *echo.HTTPError {
	status, body := httperr.MapHTTP(err)
	return echo.NewHTTPError(status, body)
}

func apiError(msg, code string) map[string]string {
	return map[string]string(httperr.Payload(code, msg))
}

func mapWSSubscribeError(err error) (code string, internal bool) {
	return httperr.MapWSSubscribe(err)
}
