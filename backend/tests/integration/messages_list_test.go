//go:build integration

package integration

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"crowdfund-app/model"
)

func TestListMessageThreads(t *testing.T) {
	var (
		e     = setupApp(t)
		gdb   = testDB(t)
		buyer *model.User
		tok   string
		err   error
		rec   *httptest.ResponseRecorder
	)
	buyer = seedUser(t, gdb, seedOpts{Role: model.UserRoleBuyer})
	tok, err = issueToken(buyer, true)
	if err != nil {
		t.Fatal(err)
	}
	rec = httpGET(e, "/messages?page=1&per_page=20&messages_per_thread=50", tok)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
