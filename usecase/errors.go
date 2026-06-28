package usecase

import (
	"errors"

	"crowdfund-app/entity"
)

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
	ErrStripeUnavailable  = errors.New("stripe unavailable")
	ErrAlreadySupporting  = errors.New("already supporting")
	ErrPendingCheckout    = errors.New("pending checkout exists")
	ErrTokenRevoked       = errors.New("token revoked")
	ErrUnsupportedMedia   = errors.New("unsupported media type")
	ErrFileTooLarge       = errors.New("file too large")
	ErrBadKYC             = errors.New("bad seller kyc")
	ErrBatchJobFailed     = errors.New("batch job failed")
	ErrFavoriteExists     = errors.New("favorite already exists")
	ErrReportExists       = errors.New("report already exists")
)

func mapBuyerRegEntityErr(err error) error {
	switch {
	case errors.Is(err, entity.ErrBadEmail):
		return ErrInvalidEmail
	case errors.Is(err, entity.ErrWeakPassword):
		return ErrWeakPassword
	default:
		return err
	}
}

func mapSellerKYCEntityErr(err error) error {
	if errors.Is(err, entity.ErrBadKYC) {
		return ErrBadKYC
	}
	return err
}

func mapProjectEntityErr(err error) error {
	switch {
	case errors.Is(err, entity.ErrBadTitle),
		errors.Is(err, entity.ErrBadProject),
		errors.Is(err, entity.ErrBadDuration),
		errors.Is(err, entity.ErrBadMembership),
		errors.Is(err, entity.ErrBadPrice),
		errors.Is(err, entity.ErrBadImages):
		return ErrBadRequest
	default:
		return err
	}
}

func mapMessageEntityErr(err error) error {
	if errors.Is(err, entity.ErrBadMessage) {
		return ErrBadRequest
	}
	return err
}
