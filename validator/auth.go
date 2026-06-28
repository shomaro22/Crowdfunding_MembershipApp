package validator

import (
	"time"

	"crowdfund-app/entity"
)

func ValidateEmail(email string) (string, error) {
	return entity.ValidateEmail(email)
}

func ValidatePassword(password string) error {
	return entity.ValidatePassword(password)
}

func ValidateBuyerReg(email, password string) (string, error) {
	return entity.ValidateBuyerReg(email, password)
}

func ValidateSellerKYC(fullName string, birthDate time.Time, address string, photoPresent bool) error {
	return entity.ValidateSellerKYC(fullName, birthDate, address, photoPresent)
}
