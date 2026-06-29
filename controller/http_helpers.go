package controller

import (
	"context"
	"net/http"
	"strconv"

	"crowdfund-app/middleware"
	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func reqCtx(c echo.Context) context.Context {
	return c.Request().Context()
}

func requireUserID(c echo.Context) (uuid.UUID, error) {
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

func bindBody(c echo.Context, dst interface{}) error {
	if err := c.Bind(dst); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid request body", "BAD_REQUEST"))
	}
	return nil
}

func pathUUID(c echo.Context, param, badMsg string) (uuid.UUID, error) {
	var (
		id  uuid.UUID
		err error
	)
	id, err = uuid.Parse(c.Param(param))
	if err != nil {
		return uuid.Nil, echo.NewHTTPError(http.StatusBadRequest, apiError(badMsg, "BAD_REQUEST"))
	}
	return id, nil
}

func queryPage(c echo.Context) (limit, offset int) {
	limit, _ = strconv.Atoi(c.QueryParam("limit"))
	offset, _ = strconv.Atoi(c.QueryParam("offset"))
	return limit, offset
}

func queryOptionalString(c echo.Context, key string) *string {
	var (
		s   string
		out *string
	)
	s = c.QueryParam(key)
	if s == "" {
		return nil
	}
	out = &s
	return out
}

type viewerCtx struct {
	UserID *uuid.UUID
	Role   string
}

func optionalViewer(c echo.Context) viewerCtx {
	var (
		v   viewerCtx
		uid uuid.UUID
		ok  bool
	)
	if uid, ok = middleware.AuthUserID(c); ok {
		v.UserID = &uid
		if role, roleOK := middleware.AuthUserRole(c); roleOK {
			v.Role = string(role)
		}
	}
	return v
}

func jsonOK(c echo.Context, body interface{}) error {
	return c.JSON(http.StatusOK, body)
}

func jsonCreated(c echo.Context, body interface{}) error {
	return c.JSON(http.StatusCreated, body)
}

func noContent(c echo.Context) error {
	return c.NoContent(http.StatusOK)
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
