package controller

import (
	"errors"
	"net/http"

	"crowdfund-app/usecase"

	"github.com/labstack/echo/v4"
)

func mapForbidden(err error) *echo.HTTPError {
	if errors.Is(err, usecase.ErrForbidden) {
		return echo.NewHTTPError(http.StatusForbidden, apiError("forbidden", "FORBIDDEN"))
	}
	return nil
}

func mapBadRequest(err error) *echo.HTTPError {
	if errors.Is(err, usecase.ErrBadRequest) {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("bad request", "BAD_REQUEST"))
	}
	return nil
}

func mapNotFound(err error) *echo.HTTPError {
	if errors.Is(err, usecase.ErrNotFound) {
		return echo.NewHTTPError(http.StatusNotFound, apiError("not found", "NOT_FOUND"))
	}
	return nil
}

func mapConflict(err error) *echo.HTTPError {
	switch {
	case errors.Is(err, usecase.ErrPendingCheckout):
		return echo.NewHTTPError(http.StatusConflict, apiError("pending checkout exists", "CONFLICT"))
	case errors.Is(err, usecase.ErrAlreadySupporting):
		return echo.NewHTTPError(http.StatusConflict, apiError("already supporting", "CONFLICT"))
	case errors.Is(err, usecase.ErrEmailInUse):
		return echo.NewHTTPError(http.StatusConflict, apiError("email already in use", "CONFLICT"))
	case errors.Is(err, usecase.Err2FAAlreadyEnabled):
		return echo.NewHTTPError(http.StatusConflict, apiError("2fa already enabled", "2FA_ALREADY_ENABLED"))
	}
	return nil
}

func mapStripeUnavailable(err error) *echo.HTTPError {
	if errors.Is(err, usecase.ErrStripeUnavailable) {
		return echo.NewHTTPError(http.StatusServiceUnavailable, apiError("stripe unavailable", "STRIPE_UNAVAILABLE"))
	}
	return nil
}

func mapInvalidState(err error) *echo.HTTPError {
	if errors.Is(err, usecase.ErrInvalidState) {
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid state", "INVALID_STATE"))
	}
	return nil
}

func mapUseCaseError(err error) *echo.HTTPError {
	var he *echo.HTTPError
	if he = mapForbidden(err); he != nil {
		return he
	}
	if he = mapBadRequest(err); he != nil {
		return he
	}
	if he = mapNotFound(err); he != nil {
		return he
	}
	if he = mapConflict(err); he != nil {
		return he
	}
	if he = mapStripeUnavailable(err); he != nil {
		return he
	}
	if he = mapInvalidState(err); he != nil {
		return he
	}
	if errors.Is(err, usecase.ErrTokenRevoked) {
		return echo.NewHTTPError(http.StatusUnauthorized, apiError("token revoked", "TOKEN_REVOKED"))
	}
	if errors.Is(err, usecase.ErrUnsupportedMedia) {
		return echo.NewHTTPError(http.StatusUnsupportedMediaType, apiError("unsupported media type", "UNSUPPORTED_MEDIA"))
	}
	if errors.Is(err, usecase.ErrFileTooLarge) {
		return echo.NewHTTPError(http.StatusRequestEntityTooLarge, apiError("file too large", "FILE_TOO_LARGE"))
	}
	return mapAuthError(err)
}
