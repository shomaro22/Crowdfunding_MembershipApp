package model

import "time"

func CanMessage(supporter ProjectSupporter, now time.Time) error {
	if !supporter.HasBenefits(now) {
		return ErrNoBenefits
	}
	return nil
}
