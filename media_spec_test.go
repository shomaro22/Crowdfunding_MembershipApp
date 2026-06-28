//go:build integration

package main

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
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
	e = setupSecurityApp(t)
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
	e = setupSecurityApp(t)
	gdb = openSecurityDB(t)
	seller = seedSecurityUser(t, gdb, seedOpts{Role: entity.UserRoleSeller})
	tok, _ = issueSecurityToken(seller, true)

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
	e = setupSecurityApp(t)
	gdb = openSecurityDB(t)
	seller = seedSecurityUser(t, gdb, seedOpts{Role: entity.UserRoleSeller})
	tok, _ = issueSecurityToken(seller, true)
	rec = mediaUploadPOST(e, tok, []byte("not an image"), "test.txt")
	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status=%d want 415", rec.Code)
	}
}

func mediaUploadPOST(e *echo.Echo, token string, data []byte, filename string) *httptest.ResponseRecorder {
	var (
		body   bytes.Buffer
		writer *multipart.Writer
		part   io.Writer
		rec    *httptest.ResponseRecorder
		req    *http.Request
		err    error
	)
	writer = multipart.NewWriter(&body)
	part, err = writer.CreateFormFile("file", filename)
	if err != nil {
		panic(err)
	}
	_, _ = part.Write(data)
	_ = writer.Close()
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/media/images", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	e.ServeHTTP(rec, req)
	return rec
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
