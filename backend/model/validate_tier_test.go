package model

import (
	"testing"
	"time"
)

func TestValidateMembershipTierCombo(t *testing.T) {
	var err error

	err = ValidateMembershipTierCombo(map[MembershipType]bool{MembershipBronze: true})
	if err != nil {
		t.Fatalf("bronze only: %v", err)
	}
	err = ValidateMembershipTierCombo(map[MembershipType]bool{
		MembershipBronze: true,
		MembershipSilver: true,
	})
	if err != nil {
		t.Fatalf("bronze+silver: %v", err)
	}
	err = ValidateMembershipTierCombo(map[MembershipType]bool{
		MembershipBronze: true,
		MembershipSilver: true,
		MembershipGold:   true,
	})
	if err != nil {
		t.Fatalf("bronze+silver+gold: %v", err)
	}
	err = ValidateMembershipTierCombo(map[MembershipType]bool{MembershipSilver: true})
	if err != ErrBadMembership {
		t.Fatalf("silver without bronze: got %v", err)
	}
	err = ValidateMembershipTierCombo(map[MembershipType]bool{
		MembershipBronze: true,
		MembershipGold:   true,
	})
	if err != ErrBadMembership {
		t.Fatalf("gold without silver: got %v", err)
	}
}

func TestValidatePriceAndTitle(t *testing.T) {
	var err error

	if err = ValidatePrice(100); err != nil {
		t.Fatalf("valid price: %v", err)
	}
	if err = ValidatePrice(0); err != ErrBadPrice {
		t.Fatalf("zero price: got %v", err)
	}
	if err = ValidateTitle("My Project"); err != nil {
		t.Fatalf("valid title: %v", err)
	}
	if err = ValidateTitle(""); err != ErrBadTitle {
		t.Fatalf("empty title: got %v", err)
	}
}

func TestValidateMessage(t *testing.T) {
	var err error

	if err = ValidateMessage("hello"); err != nil {
		t.Fatalf("valid message: %v", err)
	}
	if err = ValidateMessage(""); err != ErrBadMessage {
		t.Fatalf("empty message: got %v", err)
	}
}

func TestValidateProject(t *testing.T) {
	var (
		start = time.Now().UTC().Add(24 * time.Hour)
		end   = start.AddDate(0, 2, 0)
		err   error
	)

	err = ValidateProject("Title", "Description", 10000, DurationLimited, start, &end, []PlanInput{
		{MembershipType: MembershipBronze, Price: 500, Benefits: "Bronze perks"},
	})
	if err != nil {
		t.Fatalf("valid project: %v", err)
	}
	err = ValidateProject("", "Description", 10000, DurationLimited, start, &end, []PlanInput{
		{MembershipType: MembershipBronze, Price: 500, Benefits: "Bronze perks"},
	})
	if err != ErrBadTitle {
		t.Fatalf("empty title: got %v", err)
	}
}
