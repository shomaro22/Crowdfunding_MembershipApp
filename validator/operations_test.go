package validator

import (
	"testing"
	"time"

	"crowdfund-app/entity"

	"github.com/google/uuid"
)

func TestClampTimeRangeRejectsInverted(t *testing.T) {
	var (
		start time.Time
		end   time.Time
		err   error
	)
	start = time.Now().UTC()
	end = start.Add(-time.Hour)
	_, _, err = ClampTimeRange(start, end, MaxOperationsRange)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestClampTimeRangeRejectsTooWide(t *testing.T) {
	var (
		start time.Time
		end   time.Time
		err   error
	)
	start = time.Now().UTC().Add(-100 * 24 * time.Hour)
	end = time.Now().UTC()
	_, _, err = ClampTimeRange(start, end, MaxOperationsRange)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateActivityLogsRejectsInvalidAction(t *testing.T) {
	var (
		bad string
		err error
	)
	bad = "not.a.real.action"
	_, err = ValidateActivityLogs(ActivityLogsInput{Action: &bad})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateActivityLogsRejectsInvalidIP(t *testing.T) {
	var (
		ip  string
		err error
	)
	ip = "not-an-ip"
	_, err = ValidateActivityLogs(ActivityLogsInput{IP: &ip})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateBatchRunsRejectsInvalidJobName(t *testing.T) {
	var (
		job string
		err error
	)
	job = "reconcile_stripe"
	_, err = ValidateBatchRuns(BatchRunsInput{JobName: &job})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateBatchJobNameAcceptsSupported(t *testing.T) {
	var (
		name string
		err  error
	)
	name, err = ValidateBatchJobName(entity.PurgeTokens)
	if err != nil {
		t.Fatal(err)
	}
	if name != entity.PurgeTokens {
		t.Fatalf("name=%q", name)
	}
}

func TestValidateBatchJobNameRejectsEmpty(t *testing.T) {
	_, err := ValidateBatchJobName("  ")
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateActivityLogsNormalizesPagination(t *testing.T) {
	var (
		out ActivityLogsNormalized
		err error
	)
	out, err = ValidateActivityLogs(ActivityLogsInput{Limit: 0, Offset: 0})
	if err != nil {
		t.Fatal(err)
	}
	if out.Limit != DefaultPageLimit {
		t.Fatalf("limit=%d want %d", out.Limit, DefaultPageLimit)
	}
}

func TestValidateActivityLogsPreservesUserID(t *testing.T) {
	var (
		id  uuid.UUID
		out ActivityLogsNormalized
		err error
	)
	id = uuid.New()
	out, err = ValidateActivityLogs(ActivityLogsInput{UserID: &id})
	if err != nil {
		t.Fatal(err)
	}
	if out.UserID == nil || *out.UserID != id {
		t.Fatal("user id not preserved")
	}
}

func TestIsSupportedBatchJob(t *testing.T) {
	if !IsSupportedBatchJob(entity.ApplyPlans) {
		t.Fatal("apply_plans should be supported")
	}
	if IsSupportedBatchJob("expire_projects") {
		t.Fatal("expire_projects should not be supported")
	}
}

func TestValidateBatchRunsAcceptsValidStatus(t *testing.T) {
	var (
		status string
		out    BatchRunsNormalized
		err    error
	)
	status = string(entity.BatchSuccess)
	out, err = ValidateBatchRuns(BatchRunsInput{Status: &status})
	if err != nil {
		t.Fatal(err)
	}
	if out.Status == nil || *out.Status != entity.BatchSuccess {
		t.Fatalf("status=%v", out.Status)
	}
}

func TestValidateBatchRunsRejectsInvalidStatus(t *testing.T) {
	var (
		status string
		err    error
	)
	status = "UNKNOWN"
	_, err = ValidateBatchRuns(BatchRunsInput{Status: &status})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestIPRuleAcceptsIPv4(t *testing.T) {
	if err := ipAddressRule().Validate("203.0.113.1"); err != nil {
		t.Fatal(err)
	}
}

func TestIPRuleRejectsGarbage(t *testing.T) {
	if err := ipAddressRule().Validate("x"); err == nil {
		t.Fatal("expected error")
	}
}
