package usecase

import (
	"testing"
	"time"

	"crowdfund-app/entity"

	"github.com/google/uuid"
)

func TestKonbiniFirstPeriod(t *testing.T) {
	var (
		paid = time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
		got  periodRange
	)
	got = konbiniFirstPeriod(paid)
	if !got.start.Equal(paid) {
		t.Fatalf("start: got %v", got.start)
	}
	if !got.end.Equal(paid.AddDate(0, 0, supportPeriodDays)) {
		t.Fatalf("end: got %v", got.end)
	}
}

func TestKonbiniRenewalPeriod_FromPeriodEnd(t *testing.T) {
	var (
		end  = time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
		paid = time.Date(2026, 3, 28, 10, 0, 0, 0, time.UTC)
		s    = &entity.ProjectSupporter{CurrentPeriodEnd: end}
		got  periodRange
	)
	got = konbiniRenewalPeriod(s, paid)
	if !got.start.Equal(end) {
		t.Fatalf("start: got %v want %v", got.start, end)
	}
	if !got.end.Equal(end.AddDate(0, 0, supportPeriodDays)) {
		t.Fatalf("end: got %v", got.end)
	}
}

func TestKonbiniRenewalPeriod_LatePayment(t *testing.T) {
	var (
		end  = time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		paid = time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
		s    = &entity.ProjectSupporter{CurrentPeriodEnd: end}
		got  periodRange
	)
	got = konbiniRenewalPeriod(s, paid)
	if !got.start.Equal(paid) {
		t.Fatalf("start: got %v want %v", got.start, paid)
	}
}

func TestParseCheckoutPurpose(t *testing.T) {
	if parseCheckoutPurpose(map[string]string{}) != checkoutFirstSupport {
		t.Fatal("default should be first_support")
	}
	if parseCheckoutPurpose(map[string]string{"checkout_purpose": "konbini_renewal"}) != checkoutKonbiniRenewal {
		t.Fatal("expected renewal")
	}
	_ = uuid.Nil
}
