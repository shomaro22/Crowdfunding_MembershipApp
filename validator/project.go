package validator

import (
	"time"

	"crowdfund-app/entity"
)

func ValidateTitle(title string) error {
	return entity.ValidateTitle(title)
}

func ValidateDuration(duration entity.DurationType, start time.Time, end *time.Time) error {
	return entity.ValidateDuration(duration, start, end)
}

func ValidateProject(title, description string, targetAmount int64, duration entity.DurationType, start time.Time, end *time.Time, plans []entity.PlanInput) error {
	return entity.ValidateProject(title, description, targetAmount, duration, start, end, plans)
}

func ValidateProjectImages(images []entity.ImageInput) error {
	return entity.ValidateProjectImages(images)
}
