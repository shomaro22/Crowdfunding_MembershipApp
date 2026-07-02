package model

import "testing"

func TestEnumValidValues(t *testing.T) {
	type enumCase struct {
		name    string
		valid   func() bool
		invalid func() bool
	}

	cases := []enumCase{
		{"AccountStatus", func() bool { return AccountStatusActive.Valid() }, func() bool { return AccountStatus("X").Valid() }},
		{"SellerStatus", func() bool { return SellerStatusActive.Valid() }, func() bool { return SellerStatus("X").Valid() }},
		{"AdminRole", func() bool { return AdminRoleSuperAdmin.Valid() }, func() bool { return AdminRole("X").Valid() }},
		{"OAuthProvider", func() bool { return OAuthProviderGoogle.Valid() }, func() bool { return OAuthProvider("x").Valid() }},
		{"ProjectStatus", func() bool { return ProjectStatusActive.Valid() }, func() bool { return ProjectStatus("X").Valid() }},
		{"DurationType", func() bool { return DurationLimited.Valid() }, func() bool { return DurationType("X").Valid() }},
		{"MembershipType", func() bool { return MembershipGold.Valid() }, func() bool { return MembershipType("X").Valid() }},
		{"SupporterStatus", func() bool { return SupporterStatusExpired.Valid() }, func() bool { return SupporterStatus("X").Valid() }},
		{"BillingMode", func() bool { return BillingModeKonbini.Valid() }, func() bool { return BillingMode("X").Valid() }},
		{"PurchaseStatus", func() bool { return PurchaseStatusRefunded.Valid() }, func() bool { return PurchaseStatus("X").Valid() }},
		{"PaymentMethod", func() bool { return PaymentMethodApplePay.Valid() }, func() bool { return PaymentMethod("X").Valid() }},
		{"RegStatus", func() bool { return RegApproved.Valid() }, func() bool { return RegStatus("X").Valid() }},
		{"ReportStatus", func() bool { return ReportStatusResolved.Valid() }, func() bool { return ReportStatus("X").Valid() }},
		{"ReportTargetType", func() bool { return ReportTargetUser.Valid() }, func() bool { return ReportTargetType("X").Valid() }},
		{"BatchStatus", func() bool { return BatchSuccess.Valid() }, func() bool { return BatchStatus("X").Valid() }},
	}

	var i int
	var c enumCase
	for i = range cases {
		c = cases[i]
		if !c.valid() {
			t.Fatalf("%s: expected valid enum", c.name)
		}
		if c.invalid() {
			t.Fatalf("%s: expected invalid enum", c.name)
		}
	}
}

func TestJSONMetadataHelpers(t *testing.T) {
	var (
		step     JSONMetadata
		provider JSONMetadata
	)

	step = StepMeta("2fa_ok")
	if step == nil {
		t.Fatal("expected step meta")
	}
	provider = ProviderMeta("google")
	if provider == nil {
		t.Fatal("expected provider meta")
	}
}

func TestUserNeedsTOTP(t *testing.T) {
	var (
		admin User
		buyer User
	)

	admin = User{Role: UserRoleAdmin, AccountStatus: AccountStatusActive}
	if !admin.MustSetup2FA() {
		t.Fatal("admin must setup 2fa")
	}
	if admin.NeedsTOTP() {
		t.Fatal("admin without totp enabled should not need totp at login yet")
	}
	admin.TOTPEnabled = true
	if !admin.NeedsTOTP() {
		t.Fatal("admin with totp enabled needs challenge")
	}

	buyer = User{Role: UserRoleBuyer, AccountStatus: AccountStatusActive}
	if buyer.MustSetup2FA() || buyer.NeedsTOTP() {
		t.Fatal("buyer should not require 2fa")
	}
}
