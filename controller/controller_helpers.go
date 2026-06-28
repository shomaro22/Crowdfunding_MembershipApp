package controller

import (
	"net/http"

	"crowdfund-app/middleware"
	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func CurrentUserID(c echo.Context) (uuid.UUID, error) {
	var (
		uid uuid.UUID
		ok  bool
	)
	uid, ok = middleware.AuthUserID(c)
	if !ok {
		return uuid.Nil, echo.NewHTTPError(http.StatusUnauthorized, apiError("unauthorized", "UNAUTHORIZED"))
	}
	return uid, nil
}

func requestMeta(c echo.Context) usecase.RequestMeta {
	var reqID string
	reqID, _ = c.Get(middleware.CtxRequestID).(string)
	return usecase.RequestMeta{
		IPAddress: c.RealIP(),
		UserAgent: c.Request().UserAgent(),
		RequestID: reqID,
	}
}
