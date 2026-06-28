//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	neturl "net/url"
	"testing"

	"crowdfund-app/entity"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func TestMediaUploadUnauthorized(t *testing.T) {
	var (
		e   *echo.Echo
		rec *httptest.ResponseRecorder
	)
	e = setupApp(t)
	rec = mediaUploadPOST(e, "", minimalPNGBytes(), "test.png")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want 401", rec.Code)
	}
}

func TestMediaUploadAndFetch(t *testing.T) {
	var (
		e      *echo.Echo
		gdb    *gorm.DB
		seller *entity.User
		tok    string
		rec    *httptest.ResponseRecorder
		body   map[string]any
		urlStr string
		parsed *neturl.URL
		err    error
	)
	e = setupApp(t)
	gdb = openDB(t)
	seller = seedUser(t, gdb, seedOpts{Role: entity.UserRoleSeller})
	tok, _ = issueToken(seller, true)

	rec = mediaUploadPOST(e, tok, minimalPNGBytes(), "test.png")
	if rec.Code != http.StatusCreated {
		t.Fatalf("upload status=%d body=%s", rec.Code, rec.Body.String())
	}
	if err = json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	urlStr, _ = body["url"].(string)
	if urlStr == "" {
		t.Fatal("missing url")
	}
	parsed, err = neturl.Parse(urlStr)
	if err != nil {
		t.Fatal(err)
	}

	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, parsed.Path, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("fetch status=%d want 200 path=%s", rec.Code, parsed.Path)
	}
}

func TestMediaUploadUnsupportedType(t *testing.T) {
	var (
		e      *echo.Echo
		gdb    *gorm.DB
		seller *entity.User
		tok    string
		rec    *httptest.ResponseRecorder
	)
	e = setupApp(t)
	gdb = openDB(t)
	seller = seedUser(t, gdb, seedOpts{Role: entity.UserRoleSeller})
	tok, _ = issueToken(seller, true)
	rec = mediaUploadPOST(e, tok, []byte("not an image"), "test.txt")
	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status=%d want 415", rec.Code)
	}
}

func minimalPNGBytes() []byte {
	return []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
		0x89, 0x00, 0x00, 0x00, 0x0a, 0x49, 0x44, 0x41,
		0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00,
		0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae,
		0x42, 0x60, 0x82,
	}
}
