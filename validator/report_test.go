package validator

import (
	"testing"

	"crowdfund-app/entity"

	"github.com/google/uuid"
)

func TestValidateCreateReport(t *testing.T) {
	var (
		pid uuid.UUID
		out CreateReportNormalized
		err error
	)
	pid = uuid.New()
	out, err = ValidateCreateReport(CreateReportInput{
		TargetType: "project",
		TargetID:   pid,
		Reason:     "spam",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.TargetType != entity.ReportTargetProject {
		t.Fatalf("target_type=%q", out.TargetType)
	}
	_, err = ValidateCreateReport(CreateReportInput{
		TargetType: "INVALID",
		TargetID:   pid,
		Reason:     "spam",
	})
	if err == nil {
		t.Fatal("expected invalid target type error")
	}
}

func TestValidateAdminReportUpdate(t *testing.T) {
	var (
		status entity.ReportStatus
		err    error
	)
	status, _, err = ValidateAdminReportUpdate(AdminReportUpdateInput{Status: "resolved"})
	if err != nil || status != entity.ReportStatusResolved {
		t.Fatalf("status=%q err=%v", status, err)
	}
	_, _, err = ValidateAdminReportUpdate(AdminReportUpdateInput{Status: "bad"})
	if err == nil {
		t.Fatal("expected error")
	}
}
