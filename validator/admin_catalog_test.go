package validator

import (
	"testing"

	"crowdfund-app/entity"
)

func TestValidateAdminUserList(t *testing.T) {
	var (
		out AdminUserListNormalized
		err error
	)
	out, err = ValidateAdminUserList(AdminUserListInput{Role: "buyer", Limit: 10})
	if err != nil || out.Role == nil || *out.Role != entity.UserRoleBuyer {
		t.Fatalf("err=%v role=%v", err, out.Role)
	}
	_, err = ValidateAdminUserList(AdminUserListInput{Role: "INVALID"})
	if err == nil {
		t.Fatal("expected invalid role error")
	}
}

func TestValidateAdminPurchaseStatus(t *testing.T) {
	var (
		st  entity.PurchaseStatus
		err error
	)
	st, err = ValidateAdminPurchaseStatus("PAID")
	if err != nil || st != entity.PurchaseStatusPaid {
		t.Fatalf("err=%v st=%q", err, st)
	}
	_, err = ValidateAdminPurchaseStatus("PENDING")
	if err == nil {
		t.Fatal("expected error for PENDING")
	}
}
