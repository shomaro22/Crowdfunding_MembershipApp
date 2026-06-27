package apierr

import (
	"errors"
	"net/http"

	"crowdfund-app/usecase"
)

type Body map[string]string

func Payload(code, msg string) Body {
	return Body{"error": msg, "code": code}
}

func Unauthorized() Body {
	return Payload("UNAUTHORIZED", "unauthorized")
}

func IsTokenAuthFailure(err error) bool {
	return errors.Is(err, usecase.ErrInvalidToken) ||
		errors.Is(err, usecase.ErrForbidden) ||
		errors.Is(err, usecase.ErrInactiveUser)
}

func MapSession(err error) (int, Body) {
	switch {
	case errors.Is(err, usecase.ErrInactiveUser):
		return http.StatusForbidden, Payload("USER_INACTIVE", "user inactive")
	case errors.Is(err, usecase.ErrTokenRevoked):
		return http.StatusUnauthorized, Payload("TOKEN_REVOKED", "token revoked")
	default:
		return http.StatusInternalServerError, Payload("INTERNAL_ERROR", "internal error")
	}
}

func MapWSSubscribe(err error) (code string, internal bool) {
	switch {
	case errors.Is(err, usecase.ErrForbidden):
		return "FORBIDDEN_NOT_MEMBER", false
	case errors.Is(err, usecase.ErrNotFound):
		return "INVALID_PROJECT_ID", false
	default:
		return "INTERNAL", true
	}
}

func MapHTTP(err error) (int, Body) {
	switch {
	case errors.Is(err, usecase.ErrForbidden):
		return http.StatusForbidden, Payload("FORBIDDEN", "forbidden")
	case errors.Is(err, usecase.ErrInvalidCredentials):
		return http.StatusUnauthorized, Payload("INVALID_CREDENTIALS", "invalid credentials")
	case errors.Is(err, usecase.ErrInvalidToken):
		return http.StatusUnauthorized, Payload("INVALID_TOKEN", "invalid token")
	case errors.Is(err, usecase.ErrInvalid2FACode):
		return http.StatusUnauthorized, Payload("INVALID_2FA_CODE", "invalid 2fa code")
	case errors.Is(err, usecase.ErrTokenRevoked):
		return http.StatusUnauthorized, Payload("TOKEN_REVOKED", "token revoked")
	case errors.Is(err, usecase.ErrInactiveUser):
		return http.StatusForbidden, Payload("USER_INACTIVE", "user inactive")
	case errors.Is(err, usecase.ErrRequires2FA):
		return http.StatusForbidden, Payload("2FA_REQUIRED", "2fa required")
	case errors.Is(err, usecase.Err2FANotConfigured):
		return http.StatusForbidden, Payload("2FA_NOT_CONFIGURED", "2fa not configured")
	case errors.Is(err, usecase.ErrOAuthForbidden):
		return http.StatusForbidden, Payload("OAUTH_FORBIDDEN", "oauth forbidden")
	case errors.Is(err, usecase.ErrBadRequest):
		return http.StatusBadRequest, Payload("BAD_REQUEST", "bad request")
	case errors.Is(err, usecase.ErrBadKYC):
		return http.StatusBadRequest, Payload("BAD_SELLER_KYC", "invalid seller kyc metadata")
	case errors.Is(err, usecase.ErrWeakPassword):
		return http.StatusBadRequest, Payload("WEAK_PASSWORD", "weak password")
	case errors.Is(err, usecase.ErrInvalidEmail):
		return http.StatusBadRequest, Payload("INVALID_EMAIL", "invalid email")
	case errors.Is(err, usecase.ErrNotFound):
		return http.StatusNotFound, Payload("NOT_FOUND", "not found")
	case errors.Is(err, usecase.ErrPendingCheckout):
		return http.StatusConflict, Payload("CONFLICT", "pending checkout exists")
	case errors.Is(err, usecase.ErrAlreadySupporting):
		return http.StatusConflict, Payload("CONFLICT", "already supporting")
	case errors.Is(err, usecase.ErrEmailInUse):
		return http.StatusConflict, Payload("CONFLICT", "email already in use")
	case errors.Is(err, usecase.Err2FAAlreadyEnabled):
		return http.StatusConflict, Payload("2FA_ALREADY_ENABLED", "2fa already enabled")
	case errors.Is(err, usecase.ErrStripeUnavailable):
		return http.StatusServiceUnavailable, Payload("STRIPE_UNAVAILABLE", "stripe unavailable")
	case errors.Is(err, usecase.ErrInvalidState):
		return http.StatusBadRequest, Payload("INVALID_STATE", "invalid state")
	case errors.Is(err, usecase.ErrOAuthUnavailable):
		return http.StatusServiceUnavailable, Payload("OAUTH_UNAVAILABLE", "oauth unavailable")
	case errors.Is(err, usecase.ErrUnsupportedMedia):
		return http.StatusUnsupportedMediaType, Payload("UNSUPPORTED_MEDIA", "unsupported media type")
	case errors.Is(err, usecase.ErrFileTooLarge):
		return http.StatusRequestEntityTooLarge, Payload("FILE_TOO_LARGE", "file too large")
	default:
		return http.StatusInternalServerError, Payload("INTERNAL_ERROR", "internal error")
	}
}
