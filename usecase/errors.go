package usecase

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInactiveUser       = errors.New("user inactive")
	ErrEmailInUse         = errors.New("email already in use")
	ErrWeakPassword       = errors.New("weak password")
	ErrInvalidEmail       = errors.New("invalid email")
	ErrInvalidToken       = errors.New("invalid token")
	ErrNotFound           = errors.New("not found")
	ErrRequires2FA        = errors.New("2fa required")
	Err2FANotConfigured   = errors.New("2fa not configured")
	ErrInvalidState       = errors.New("invalid state")
	ErrBadRequest         = errors.New("bad request")
	ErrInvalid2FACode     = errors.New("invalid 2fa code")
	Err2FAAlreadyEnabled  = errors.New("2fa already enabled")
	ErrOAuthUnavailable   = errors.New("oauth unavailable")
	ErrOAuthForbidden     = errors.New("oauth forbidden")
	ErrForbidden          = errors.New("forbidden")
)
