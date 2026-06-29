package controller

import (
	"crowdfund-app/apierr"

	"github.com/labstack/echo/v4"
)

func mapUseCaseError(err error) *echo.HTTPError {
	var (
		status int
		body   apierr.Body
	)
	status, body = apierr.MapHTTP(err)
	return echo.NewHTTPError(status, body)
}

func apiError(msg, code string) map[string]string {
	return map[string]string(apierr.Payload(code, msg))
}
