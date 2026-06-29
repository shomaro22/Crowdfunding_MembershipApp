package validator

import (
	"testing"

	"github.com/google/uuid"
)

func TestValidateFavoriteToggle(t *testing.T) {
	var (
		pid uuid.UUID
		act string
		err error
	)
	pid = uuid.New()
	_, _, err = ValidateFavoriteToggle(FavoriteToggleInput{ProjectID: pid, Action: FavoriteActionAdd})
	if err != nil {
		t.Fatal(err)
	}
	_, act, err = ValidateFavoriteToggle(FavoriteToggleInput{ProjectID: pid, Action: " remove "})
	if err != nil || act != FavoriteActionRemove {
		t.Fatalf("action=%q err=%v", act, err)
	}
	_, _, err = ValidateFavoriteToggle(FavoriteToggleInput{ProjectID: pid, Action: "toggle"})
	if err == nil {
		t.Fatal("expected error for invalid action")
	}
}
