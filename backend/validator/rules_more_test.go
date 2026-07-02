package validator

import (
	"testing"
	"time"
)

func TestPaginateDefaults(t *testing.T) {
	var (
		limit  int
		offset int
		err    error
	)

	limit, offset, err = Paginate(0, 0)
	if err != nil || limit != DefaultPageLimit || offset != 0 {
		t.Fatalf("paginate default: limit=%d offset=%d err=%v", limit, offset, err)
	}
	limit, _, err = Paginate(500, 0)
	if err != nil || limit != MaxPageLimit {
		t.Fatalf("paginate cap: limit=%d err=%v", limit, err)
	}
	_, _, err = Paginate(10, -1)
	if err == nil {
		t.Fatal("expected negative offset error")
	}
}

func TestClampTimeRangeRejectsInvertedRange(t *testing.T) {
	var (
		start = time.Now().UTC()
		end   = start.Add(-time.Hour)
		_, _, err = ClampTimeRange(start, end, MaxOperationsRange)
	)
	if err == nil {
		t.Fatal("expected inverted range error")
	}
}
