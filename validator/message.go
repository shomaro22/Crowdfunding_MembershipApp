package validator

import "crowdfund-app/entity"

func ValidateMessage(content string) error {
	return entity.ValidateMessage(content)
}
