package validator_test

import (
	"testing"
	"time"

	"crowdfund-app/model"
	"crowdfund-app/validator"
)

func TestValidateAuthWrappers(t *testing.T) {
	var (
		email string
		err   error
		birth time.Time
	)

	email, err = validator.ValidateEmail("buyer@example.com")
	if err != nil || email != "buyer@example.com" {
		t.Fatalf("email: %v %q", err, email)
	}
	_, err = validator.ValidateEmail("bad")
	if err == nil {
		t.Fatal("expected invalid email")
	}
	if err = validator.ValidatePassword("longenough"); err != nil {
		t.Fatalf("password: %v", err)
	}
	email, err = validator.ValidateBuyerReg("buyer@example.com", "longenough")
	if err != nil || email != "buyer@example.com" {
		t.Fatalf("buyer reg: %v", err)
	}
	birth = time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)
	if err = validator.ValidateSellerKYC("山田", birth, "東京都", true); err != nil {
		t.Fatalf("seller kyc: %v", err)
	}
}

func TestValidateProjectWrappers(t *testing.T) {
	var (
		start = time.Now().UTC().Add(24 * time.Hour)
		end   = start.AddDate(0, 1, 0)
		err   error
	)

	if err = validator.ValidateTitle("Title"); err != nil {
		t.Fatalf("title: %v", err)
	}
	if err = validator.ValidateDuration(model.DurationLimited, start, &end); err != nil {
		t.Fatalf("duration: %v", err)
	}
	if err = validator.ValidateMessage("hello"); err != nil {
		t.Fatalf("message: %v", err)
	}
	if err = validator.ValidateProjectImages([]model.ImageInput{
		{ImageURL: "https://example.com/a.jpg", IsMain: true},
	}); err != nil {
		t.Fatalf("images: %v", err)
	}
}
