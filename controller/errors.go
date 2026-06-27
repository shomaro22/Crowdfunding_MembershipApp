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
	switch {
	case errors.Is(err, usecase.ErrBadRequest):
		return echo.NewHTTPError(http.StatusBadRequest, apiError("bad request", "BAD_REQUEST"))
	case errors.Is(err, usecase.ErrBadKYC):
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid seller kyc metadata", "BAD_SELLER_KYC"))
	case errors.Is(err, usecase.ErrWeakPassword):
		return echo.NewHTTPError(http.StatusBadRequest, apiError("weak password", "WEAK_PASSWORD"))
	case errors.Is(err, usecase.ErrInvalidEmail):
		return echo.NewHTTPError(http.StatusBadRequest, apiError("invalid email", "INVALID_EMAIL"))
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

func mapUnauthorized(err error) *echo.HTTPError {
	switch {
	case errors.Is(err, usecase.ErrInvalidCredentials):
		return echo.NewHTTPError(http.StatusUnauthorized, apiError("invalid credentials", "INVALID_CREDENTIALS"))
	case errors.Is(err, usecase.ErrInvalidToken):
		return echo.NewHTTPError(http.StatusUnauthorized, apiError("invalid token", "INVALID_TOKEN"))
	case errors.Is(err, usecase.ErrInvalid2FACode):
		return echo.NewHTTPError(http.StatusUnauthorized, apiError("invalid 2fa code", "INVALID_2FA_CODE"))
	case errors.Is(err, usecase.ErrTokenRevoked):
		return echo.NewHTTPError(http.StatusUnauthorized, apiError("token revoked", "TOKEN_REVOKED"))
	}
	return nil
}

func mapAuthForbidden(err error) *echo.HTTPError {
	switch {
	case errors.Is(err, usecase.ErrInactiveUser):
		return echo.NewHTTPError(http.StatusForbidden, apiError("user inactive", "USER_INACTIVE"))
	case errors.Is(err, usecase.ErrRequires2FA):
		return echo.NewHTTPError(http.StatusForbidden, apiError("2fa required", "2FA_REQUIRED"))
	case errors.Is(err, usecase.Err2FANotConfigured):
		return echo.NewHTTPError(http.StatusForbidden, apiError("2fa not configured", "2FA_NOT_CONFIGURED"))
	case errors.Is(err, usecase.ErrOAuthForbidden):
		return echo.NewHTTPError(http.StatusForbidden, apiError("oauth forbidden", "OAUTH_FORBIDDEN"))
	}
	return nil
}

func mapUseCaseError(err error) *echo.HTTPError {
	var he *echo.HTTPError
	if he = mapForbidden(err); he != nil {
		return he
	}
	if he = mapUnauthorized(err); he != nil {
		return he
	}
	if he = mapAuthForbidden(err); he != nil {
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
	if errors.Is(err, usecase.ErrOAuthUnavailable) {
		return echo.NewHTTPError(http.StatusServiceUnavailable, apiError("oauth unavailable", "OAUTH_UNAVAILABLE"))
	}
	if errors.Is(err, usecase.ErrUnsupportedMedia) {
		return echo.NewHTTPError(http.StatusUnsupportedMediaType, apiError("unsupported media type", "UNSUPPORTED_MEDIA"))
	}
	if errors.Is(err, usecase.ErrFileTooLarge) {
		return echo.NewHTTPError(http.StatusRequestEntityTooLarge, apiError("file too large", "FILE_TOO_LARGE"))
	}
	return echo.NewHTTPError(http.StatusInternalServerError, apiError("internal error", "INTERNAL_ERROR"))
}
