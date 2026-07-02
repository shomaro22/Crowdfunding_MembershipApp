package model

import (
	"encoding/json"
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
	var (
		statuses = []AccountStatus{AccountStatusPending, AccountStatusSuspended, AccountStatusBanned}
		i        int
		s        AccountStatus
	)

	if !AccountStatusActive.CanLogin() {
		t.Fatal("ACTIVE should allow login")
	}
	for i = range statuses {
		s = statuses[i]
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

func TestSellerIDDoc(t *testing.T) {
	var err error

	if !IDDriversLicense.Valid() {
		t.Fatal("DRIVERS_LICENSE should be valid")
	}
	if IDDocType("MY_NUMBER_CARD").Valid() {
		t.Fatal("only DRIVERS_LICENSE is in scope per §5.1.8")
	}
	if err = ValidateSellerIDDoc(IDDriversLicense); err != nil {
		t.Fatalf("expected ok: %v", err)
	}
	if err = ValidateSellerIDDoc("PASSPORT"); err != ErrBadIDDoc {
		t.Fatalf("expected ErrBadIDDoc, got %v", err)
	}
}

func TestNormalizeEmail(t *testing.T) {
	var got string

	got = NormalizeEmail("  Foo@Example.COM ")
	if got != "foo@example.com" {
		t.Fatalf("got %q", got)
	}
}

func TestValidateEmail(t *testing.T) {
	var (
		email string
		err   error
	)

	email, err = ValidateEmail("buyer@example.com")
	if err != nil || email != "buyer@example.com" {
		t.Fatalf("valid email: %v %q", err, email)
	}
	_, err = ValidateEmail("not-an-email")
	if err == nil {
		t.Fatal("expected invalid email error")
	}
}

func TestValidatePassword(t *testing.T) {
	var err error

	if err = ValidatePassword("short"); err != ErrWeakPassword {
		t.Fatalf("expected weak password, got %v", err)
	}
	if err = ValidatePassword("longenough"); err != nil {
		t.Fatalf("expected ok, got %v", err)
	}
}

func TestValidateBuyerReg(t *testing.T) {
	var (
		email string
		err   error
	)

	email, err = ValidateBuyerReg("buyer@example.com", "longenough")
	if err != nil || email != "buyer@example.com" {
		t.Fatalf("buyer reg: %v %q", err, email)
	}
	_, err = ValidateBuyerReg("bad", "longenough")
	if err == nil {
		t.Fatal("expected invalid email")
	}
}

func TestUserIsActive(t *testing.T) {
	var u User

	u = User{AccountStatus: AccountStatusActive}
	if !u.IsActive() {
		t.Fatal("ACTIVE user should be is_active=true for API")
	}
	u.AccountStatus = AccountStatusPending
	if u.IsActive() {
		t.Fatal("PENDING user should be is_active=false")
	}
}

func TestSellerLogin(t *testing.T) {
	var u User

	u = User{
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
	var (
		now     time.Time
		end     time.Time
		active  ProjectSupporter
		renewal ProjectSupporter
		expired ProjectSupporter
	)

	now = time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC)
	end = now.Add(24 * time.Hour)

	active = ProjectSupporter{
		Status:           SupporterStatusActive,
		CurrentPeriodEnd: end,
	}
	if !active.HasBenefits(now) {
		t.Fatal("active supporter within period should have benefits")
	}

	renewal = ProjectSupporter{
		Status:           SupporterStatusPendingRenewal,
		CurrentPeriodEnd: end,
	}
	if !renewal.HasBenefits(now) {
		t.Fatal("PENDING_RENEWAL within period should retain benefits per §5.3.2a")
	}

	expired = ProjectSupporter{
		Status:           SupporterStatusPendingRenewal,
		CurrentPeriodEnd: now.Add(-time.Hour),
	}
	if expired.HasBenefits(now) {
		t.Fatal("past period end should not have benefits")
	}
}

func TestActivityAction(t *testing.T) {
	var (
		actions = []ActivityAction{
			UserLogin,
			UserRegister,
			FavoriteAdd,
			PurchaseCreated,
			MessageSent,
			ReportCreated,
		}
		i int
		a ActivityAction
	)

	for i = range actions {
		a = actions[i]
		if !a.Valid() {
			t.Fatalf("action %q should be valid per §16.3", a)
		}
	}
}

func TestJSONMetadata(t *testing.T) {
	type purchaseMeta struct {
		ProjectID string  `json:"project_id"`
		Amount    float64 `json:"amount"`
	}

	var (
		raw     []byte
		err     error
		orig    JSONMetadata
		val     interface{}
		decoded JSONMetadata
		got     purchaseMeta
	)

	raw, err = json.Marshal(purchaseMeta{ProjectID: "abc", Amount: 1000})
	if err != nil {
		t.Fatal(err)
	}
	orig = JSONMetadata(raw)
	val, err = orig.Value()
	if err != nil {
		t.Fatal(err)
	}
	if err = decoded.Scan(val); err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal([]byte(decoded), &got); err != nil {
		t.Fatal(err)
	}
	if got.ProjectID != "abc" || got.Amount != 1000 {
		t.Fatalf("got %+v", got)
	}
}

func TestValidateSellerKYC(t *testing.T) {
	var (
		birth time.Time
		err   error
	)

	birth = time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)
	if err = ValidateSellerKYC("山田 太郎", birth, "東京都千代田区", true); err != nil {
		t.Fatalf("expected ok: %v", err)
	}
	if err = ValidateSellerKYC("", birth, "東京都", true); err != ErrBadKYC {
		t.Fatalf("empty name: %v", err)
	}
	if err = ValidateSellerKYC("山田", time.Time{}, "東京都", true); err != ErrBadKYC {
		t.Fatalf("zero birth date: %v", err)
	}
	if err = ValidateSellerKYC("山田", birth, "  ", true); err != ErrBadKYC {
		t.Fatalf("empty address: %v", err)
	}
	if err = ValidateSellerKYC("山田", birth, "東京都", false); err != ErrBadKYC {
		t.Fatalf("photo_present false: %v", err)
	}
}

func TestSellerRegApprove(t *testing.T) {
	var (
		front string
		back  string
		birth time.Time
		reg   SellerReg
	)

	front = "kyc/front.jpg"
	back = "kyc/back.jpg"
	birth = time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)

	reg = SellerReg{
		Status:       RegPending,
		FullName:     "山田 太郎",
		BirthDate:    birth,
		Address:      "東京都千代田区",
		PhotoPresent: true,
		IDFrontKey:   &front,
		IDBackKey:    &back,
	}
	if !reg.CanApprove() {
		t.Fatal("complete application should be approvable")
	}
	if reg.IDDocType() != IDDriversLicense {
		t.Fatal("KYC scope is drivers license only")
	}

	reg.IDBackKey = nil
	if reg.CanApprove() {
		t.Fatal("missing back image should not be approvable")
	}

	reg.IDBackKey = &back
	reg.PhotoPresent = false
	if reg.CanApprove() {
		t.Fatal("missing photo_present should not be approvable")
	}
}

func TestDurationDates(t *testing.T) {
	var (
		start time.Time
		end   time.Time
		err   error
		now   time.Time
	)

	now = time.Now().UTC()
	start = now.Add(24 * time.Hour)
	end = start.AddDate(0, 6, 0)

	if err = ValidateDuration(DurationLimited, start, &end); err != nil {
		t.Fatalf("limited with end: %v", err)
	}
	if err = ValidateDuration(DurationUnlimited, start, nil); err != nil {
		t.Fatalf("unlimited without end: %v", err)
	}
	if err = ValidateDuration(DurationUnlimited, start, &end); err == nil {
		t.Fatal("unlimited with end should fail")
	}
	if err = ValidateDuration(DurationLimited, now.Add(-48*time.Hour), &end); err == nil {
		t.Fatal("past start should fail")
	}
}

func TestValidateProjectImages(t *testing.T) {
	var err error

	if err = ValidateProjectImages(nil); err != nil {
		t.Fatalf("empty images: %v", err)
	}
	if err = ValidateProjectImages([]ImageInput{{ImageURL: "https://example.com/a.jpg", IsMain: true}}); err != nil {
		t.Fatalf("single main image: %v", err)
	}
	if err = ValidateProjectImages([]ImageInput{
		{ImageURL: "https://example.com/a.jpg", IsMain: true},
		{ImageURL: "https://example.com/b.jpg", IsMain: false},
	}); err != nil {
		t.Fatalf("two images with one main: %v", err)
	}
	if err = ValidateProjectImages([]ImageInput{{ImageURL: "https://example.com/a.jpg", IsMain: false}}); err != ErrBadImages {
		t.Fatalf("missing main image: got %v", err)
	}
	if err = ValidateProjectImages(make([]ImageInput, 11)); err != ErrBadImages {
		t.Fatalf("too many images: got %v", err)
	}
}
