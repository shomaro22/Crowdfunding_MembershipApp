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
