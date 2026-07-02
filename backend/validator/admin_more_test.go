package validator

import (
	"testing"
	"time"

	"crowdfund-app/model"

	"github.com/google/uuid"
)

func TestValidateAdminUpdateUser(t *testing.T) {
	var (
		active = true
		role   = "SELLER"
		out    AdminUpdateUserNormalized
		err    error
	)

	out, err = ValidateAdminUpdateUser(AdminUpdateUserInput{IsActive: &active})
	if err != nil || !out.IsActiveSet {
		t.Fatalf("update active: err=%v out=%+v", err, out)
	}
	out, err = ValidateAdminUpdateUser(AdminUpdateUserInput{Role: &role})
	if err != nil || !out.RoleSet || *out.Role != model.UserRoleSeller {
		t.Fatalf("update role: err=%v out=%+v", err, out)
	}
	_, err = ValidateAdminUpdateUser(AdminUpdateUserInput{})
	if err == nil {
		t.Fatal("expected empty update error")
	}
}

func TestValidateAdminPurchaseList(t *testing.T) {
	var (
		out AdminPurchaseListNormalized
		err error
	)

	out, err = ValidateAdminPurchaseList(AdminPurchaseListInput{Status: "PAID", Limit: 20})
	if err != nil || out.Status == nil || *out.Status != model.PurchaseStatusPaid {
		t.Fatalf("purchase list: err=%v out=%+v", err, out)
	}
	_, err = ValidateAdminPurchaseList(AdminPurchaseListInput{Status: "INVALID"})
	if err == nil {
		t.Fatal("expected invalid status")
	}
}

func TestValidateSellerSupporterList(t *testing.T) {
	var (
		pid = uuid.New()
		out SellerSupporterListNormalized
		err error
	)

	out, err = ValidateSellerSupporterList(SellerSupporterListInput{ProjectID: &pid, Limit: 10})
	if err != nil || out.ProjectID == nil || *out.ProjectID != pid {
		t.Fatalf("supporter list: err=%v out=%+v", err, out)
	}
}

func TestValidateProjectFull(t *testing.T) {
	var (
		start = time.Now().UTC().Add(48 * time.Hour)
		end   = start.AddDate(0, 6, 0)
		err   error
	)

	err = ValidateProject("Title", "Desc", 1000, model.DurationLimited, start, &end, []model.PlanInput{
		{MembershipType: model.MembershipBronze, Price: 100, Benefits: "b"},
		{MembershipType: model.MembershipSilver, Price: 200, Benefits: "s"},
	})
	if err != nil {
		t.Fatalf("valid project: %v", err)
	}
}
