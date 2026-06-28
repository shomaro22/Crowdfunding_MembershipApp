package validator

import (
	"strings"

	"github.com/google/uuid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const (
	FavoriteActionAdd    = "add"
	FavoriteActionRemove = "remove"
)

type FavoriteToggleInput struct {
	ProjectID uuid.UUID
	Action    string
}

func ValidateFavoriteToggle(in FavoriteToggleInput) (uuid.UUID, string, error) {
	var (
		action string
		err    error
	)
	action = strings.TrimSpace(in.Action)
	err = validation.ValidateStruct(&in,
		validation.Field(&in.ProjectID, validation.Required),
	)
	if err != nil {
		return uuid.Nil, "", err
	}
	err = validation.Validate(action, validation.Required, validation.In(FavoriteActionAdd, FavoriteActionRemove))
	if err != nil {
		return uuid.Nil, "", err
	}
	return in.ProjectID, action, nil
}
