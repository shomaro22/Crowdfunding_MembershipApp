//go:build integration

package integration

import (
	"testing"

	"crowdfund-app/repository"

	"github.com/google/uuid"
)

func TestMessageThreadSummariesRepo(t *testing.T) {
	var (
		gdb      = testDB(t)
		repo     = repository.NewMessageRepo(gdb)
		userID   = uuid.New()
		rows     []repository.MessageThreadSummary
		total    int64
		err      error
	)
	rows, total, err = repo.ListThreadSummaries(t.Context(), userID, 0, 20)
	if err != nil {
		t.Fatalf("repo err: %v", err)
	}
	t.Logf("rows=%d total=%d", len(rows), total)
}
