package validator

import (
	"time"

	"crowdfund-app/model"
)

func ValidateTitle(title string) error {
	return model.ValidateTitle(title)
}

func ValidateDuration(duration model.DurationType, start time.Time, end *time.Time) error {
	return model.ValidateDuration(duration, start, end)
}

func ValidateProject(title, description string, targetAmount int64, duration model.DurationType, start time.Time, end *time.Time, plans []model.PlanInput) error {
	return model.ValidateProject(title, description, targetAmount, duration, start, end, plans)
}

func ValidateProjectImages(images []model.ImageInput) error {
	return model.ValidateProjectImages(images)
}

func ValidateMessage(content string) error {
	return model.ValidateMessage(content)
}
