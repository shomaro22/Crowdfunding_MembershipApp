package usecase

import (
	"encoding/json"
	"testing"
)

func TestActivityMeta(t *testing.T) {
	var (
		meta = activityMeta(map[string]string{"flow": "checkout", "project_id": "abc"})
		m    map[string]string
	)
	if err := json.Unmarshal([]byte(meta), &m); err != nil {
		t.Fatal(err)
	}
	if m["flow"] != "checkout" {
		t.Fatalf("got %v", m)
	}
}
