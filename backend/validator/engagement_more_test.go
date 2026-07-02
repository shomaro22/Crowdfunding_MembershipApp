package validator

import (
	"testing"
	"time"

	"crowdfund-app/model"

	"github.com/google/uuid"
)

func TestValidateIAdminReportList(t *testing.T) {
	var (
		status = "PENDING"
		out    IAdminReportListNormalized
		err    error
	)

	out, err = ValidateIAdminReportList(IAdminReportListInput{Status: &status, Limit: 10})
	if err != nil || out.Status == nil || *out.Status != model.ReportStatusPending {
		t.Fatalf("report list: err=%v out=%+v", err, out)
	}
}

func TestValidateAuditLogsAndSecurityRange(t *testing.T) {
	var (
		out   AuditLogsNormalized
		err   error
		from  time.Time
		to    time.Time
		admin = uuid.New()
	)

	out, err = ValidateAuditLogs(AuditLogsInput{Limit: 10, AdminID: &admin})
	if err != nil || out.Limit != 10 {
		t.Fatalf("audit logs: err=%v out=%+v", err, out)
	}
	from, to, err = ValidateSecurityTimeRange(nil, nil)
	if err != nil || !to.After(from) {
		t.Fatalf("security range: err=%v from=%v to=%v", err, from, to)
	}
}
