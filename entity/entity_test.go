package entity

import (
	"testing"
	"time"
)

func TestUserRole(t *testing.T) {
	if !UserRoleBuyer.Valid() || !UserRoleSeller.Valid() || !UserRoleAdmin.Valid() {
		t.Fatal("user roles should be valid")
	}
	if UserRole("INVALID").Valid() {
		t.Fatal("invalid role should fail Valid()")
	}
}

func TestAccountLogin(t *testing.T) {
	if !AccountStatusActive.CanLogin() {
		t.Fatal("ACTIVE should allow login")
	}
	for _, s := range []AccountStatus{AccountStatusPending, AccountStatusSuspended, AccountStatusBanned} {
		if s.CanLogin() {
			t.Fatalf("%s should not allow login", s)
		}
	}
}

func TestMembershipRank(t *testing.T) {
	if MembershipBronze.Rank() >= MembershipSilver.Rank() || MembershipSilver.Rank() >= MembershipGold.Rank() {
		t.Fatal("bronze < silver < gold")
	}
	if !MembershipGold.MeetsMinimum(MembershipSilver) {
		t.Fatal("gold should meet silver minimum")
	}
	if MembershipBronze.MeetsMinimum(MembershipGold) {
		t.Fatal("bronze should not meet gold minimum")
	}
}

func TestSupporterRenewal(t *testing.T) {
	if !SupporterStatusPendingRenewal.Valid() {
		t.Fatal("PENDING_RENEWAL is in spec but missing from openapi SupporterStatus enum")
	}
}

func TestPurchaseDisputed(t *testing.T) {
	if !PurchaseStatusDisputed.Valid() || !PurchaseStatusFailed.Valid() {
		t.Fatal("DISPUTED and FAILED are required by spec §5.3")
	}
}

func TestIdentityDocBack(t *testing.T) {
	if !IDDriversLicense.NeedsBack() {
		t.Fatal("drivers license requires back")
	}
	if IDMyNumberCard.NeedsBack() {
		t.Fatal("my number card is front only")
	}
}

func TestNormalizeEmail(t *testing.T) {
	got := NormalizeEmail("  Foo@Example.COM ")
	if got != "foo@example.com" {
		t.Fatalf("got %q", got)
	}
}

func TestValidateEmail(t *testing.T) {
	email, err := ValidateEmail("buyer@example.com")
	if err != nil || email != "buyer@example.com" {
		t.Fatalf("valid email: %v %q", err, email)
	}
	if _, err := ValidateEmail("not-an-email"); err == nil {
		t.Fatal("expected invalid email error")
	}
}

func TestValidatePassword(t *testing.T) {
	if err := ValidatePassword("short"); err != ErrWeakPassword {
		t.Fatalf("expected weak password, got %v", err)
	}
	if err := ValidatePassword("longenough"); err != nil {
		t.Fatalf("expected ok, got %v", err)
	}
}

func TestUserIsActive(t *testing.T) {
	u := User{AccountStatus: AccountStatusActive}
	if !u.IsActive() {
		t.Fatal("ACTIVE user should be is_active=true for API")
	}
	u.AccountStatus = AccountStatusPending
	if u.IsActive() {
		t.Fatal("PENDING user should be is_active=false")
	}
}

func TestSellerLogin(t *testing.T) {
	u := User{
		Role:          UserRoleSeller,
		AccountStatus: AccountStatusActive,
		SellerStatus:  SellerStatusPending,
	}
	if u.CanLogin() {
		t.Fatal("seller with PENDING seller_status cannot login")
	}
	u.SellerStatus = SellerStatusActive
	if !u.CanLogin() {
		t.Fatal("approved seller should login")
	}
}

func TestSupporterBenefits(t *testing.T) {
	now := time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC)
	end := now.Add(24 * time.Hour)

	active := ProjectSupporter{
		Status:           SupporterStatusActive,
		CurrentPeriodEnd: end,
	}
	if !active.HasBenefits(now) {
		t.Fatal("active supporter within period should have benefits")
	}

	renewal := ProjectSupporter{
		Status:           SupporterStatusPendingRenewal,
		CurrentPeriodEnd: end,
	}
	if !renewal.HasBenefits(now) {
		t.Fatal("PENDING_RENEWAL within period should retain benefits per §5.3.2a")
	}

	expired := ProjectSupporter{
		Status:           SupporterStatusPendingRenewal,
		CurrentPeriodEnd: now.Add(-time.Hour),
	}
	if expired.HasBenefits(now) {
		t.Fatal("past period end should not have benefits")
	}
}

func TestActivityAction(t *testing.T) {
	actions := []ActivityAction{
		UserLogin,
		UserRegister,
		FavoriteAdd,
		PurchaseCreated,
		MessageSent,
		ReportCreated,
	}
	for _, a := range actions {
		if !a.Valid() {
			t.Fatalf("action %q should be valid per §16.3", a)
		}
	}
}

func TestJSONMetadata(t *testing.T) {
	orig := JSONMetadata{"project_id": "abc", "amount": float64(1000)}
	val, err := orig.Value()
	if err != nil {
		t.Fatal(err)
	}
	var decoded JSONMetadata
	if err := decoded.Scan(val); err != nil {
		t.Fatal(err)
	}
	if decoded["project_id"] != "abc" {
		t.Fatalf("got %v", decoded)
	}
}

func TestSellerRegApprove(t *testing.T) {
	front := "kyc/front.jpg"
	reg := SellerReg{
		Status:     RegPending,
		IDDocType:  IDMyNumberCard,
		IDFrontKey: &front,
	}
	if !reg.CanApprove() {
		t.Fatal("front-only document should be approvable with front key")
	}

	reg.IDDocType = IDDriversLicense
	if reg.CanApprove() {
		t.Fatal("drivers license without back should not be approvable")
	}
	back := "kyc/back.jpg"
	reg.IDBackKey = &back
	if !reg.CanApprove() {
		t.Fatal("drivers license with both sides should be approvable")
	}
}

func TestDurationDates(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)

	if err := ValidateDuration(DurationLimited, start, &end); err != nil {
		t.Fatalf("limited with end: %v", err)
	}
	if err := ValidateDuration(DurationUnlimited, start, nil); err != nil {
		t.Fatalf("unlimited without end: %v", err)
	}
	if err := ValidateDuration(DurationUnlimited, start, &end); err == nil {
		t.Fatal("unlimited with end should fail")
	}
}
