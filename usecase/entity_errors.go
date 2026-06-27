package usecase

import (
	"errors"

	"crowdfund-app/entity"
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
