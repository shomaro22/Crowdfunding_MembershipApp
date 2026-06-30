package controller

import (
	"errors"
	"net/http"

	"crowdfund-app/usecase"

	"github.com/labstack/echo/v4"
)

func mapUseCaseError(err error) *echo.HTTPError {
	status, body := mapUseCaseHTTP(err)
	return echo.NewHTTPError(status, body)
}

func apiError(msg, code string) map[string]string {
	return errorPayload(code, msg)
}

func errorPayload(code, msg string) map[string]string {
	return map[string]string{"error": msg, "code": code}
}

func mapWSSubscribeError(err error) (code string, internal bool) {
	switch {
	case errors.Is(err, usecase.ErrForbidden):
		return "FORBIDDEN_NOT_MEMBER", false
	case errors.Is(err, usecase.ErrNotFound):
		return "INVALID_PROJECT_ID", false
	default:
		return "INTERNAL", true
	}
}

func mapUseCaseHTTP(err error) (int, map[string]string) {
	switch {
	case errors.Is(err, usecase.ErrForbidden):
		return http.StatusForbidden, errorPayload("FORBIDDEN", "forbidden")
	case errors.Is(err, usecase.ErrInvalidCredentials):
		return http.StatusUnauthorized, errorPayload("INVALID_CREDENTIALS", "invalid credentials")
	case errors.Is(err, usecase.ErrInvalidToken):
		return http.StatusUnauthorized, errorPayload("INVALID_TOKEN", "invalid token")
	case errors.Is(err, usecase.ErrInvalid2FACode):
		return http.StatusUnauthorized, errorPayload("INVALID_2FA_CODE", "invalid 2fa code")
	case errors.Is(err, usecase.ErrTokenRevoked):
		return http.StatusUnauthorized, errorPayload("TOKEN_REVOKED", "token revoked")
	case errors.Is(err, usecase.ErrInactiveUser):
		return http.StatusForbidden, errorPayload("USER_INACTIVE", "user inactive")
	case errors.Is(err, usecase.ErrRequires2FA):
		return http.StatusForbidden, errorPayload("2FA_REQUIRED", "2fa required")
	case errors.Is(err, usecase.Err2FANotConfigured):
		return http.StatusForbidden, errorPayload("2FA_NOT_CONFIGURED", "2fa not configured")
	case errors.Is(err, usecase.ErrOAuthForbidden):
		return http.StatusForbidden, errorPayload("OAUTH_FORBIDDEN", "oauth forbidden")
	case errors.Is(err, usecase.ErrBadRequest):
		return http.StatusBadRequest, errorPayload("BAD_REQUEST", "bad request")
	case errors.Is(err, usecase.ErrBadKYC):
		return http.StatusBadRequest, errorPayload("BAD_SELLER_KYC", "invalid seller kyc metadata")
	case errors.Is(err, usecase.ErrWeakPassword):
		return http.StatusBadRequest, errorPayload("WEAK_PASSWORD", "weak password")
	case errors.Is(err, usecase.ErrInvalidEmail):
		return http.StatusBadRequest, errorPayload("INVALID_EMAIL", "invalid email")
	case errors.Is(err, usecase.ErrNotFound):
		return http.StatusNotFound, errorPayload("NOT_FOUND", "not found")
	case errors.Is(err, usecase.ErrPendingCheckout):
		return http.StatusConflict, errorPayload("CONFLICT", "pending checkout exists")
	case errors.Is(err, usecase.ErrAlreadySupporting):
		return http.StatusConflict, errorPayload("CONFLICT", "already supporting")
	case errors.Is(err, usecase.ErrEmailInUse):
		return http.StatusConflict, errorPayload("CONFLICT", "email already in use")
	case errors.Is(err, usecase.ErrFavoriteExists):
		return http.StatusConflict, errorPayload("CONFLICT", "favorite already exists")
	case errors.Is(err, usecase.ErrReportExists):
		return http.StatusConflict, errorPayload("CONFLICT", "report already exists")
	case errors.Is(err, usecase.Err2FAAlreadyEnabled):
		return http.StatusConflict, errorPayload("2FA_ALREADY_ENABLED", "2fa already enabled")
	case errors.Is(err, usecase.ErrStripeUnavailable):
		return http.StatusServiceUnavailable, errorPayload("STRIPE_UNAVAILABLE", "stripe unavailable")
	case errors.Is(err, usecase.ErrInvalidState):
		return http.StatusBadRequest, errorPayload("INVALID_STATE", "invalid state")
	case errors.Is(err, usecase.ErrOAuthUnavailable):
		return http.StatusServiceUnavailable, errorPayload("OAUTH_UNAVAILABLE", "oauth unavailable")
	case errors.Is(err, usecase.ErrUnsupportedMedia):
		return http.StatusUnsupportedMediaType, errorPayload("UNSUPPORTED_MEDIA", "unsupported media type")
	case errors.Is(err, usecase.ErrFileTooLarge):
		return http.StatusRequestEntityTooLarge, errorPayload("FILE_TOO_LARGE", "file too large")
	case errors.Is(err, usecase.ErrBatchJobFailed):
		return http.StatusUnprocessableEntity, errorPayload("BATCH_JOB_FAILED", "batch job failed")
	default:
		return http.StatusInternalServerError, errorPayload("INTERNAL_ERROR", "internal error")
	}
}
