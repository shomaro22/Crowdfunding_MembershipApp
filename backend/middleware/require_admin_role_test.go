package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"crowdfund-app/model"

	"github.com/labstack/echo/v4"
)

func TestRequireAdminRole(t *testing.T) {
	var (
		e    *echo.Echo
		req  *http.Request
		rec  *httptest.ResponseRecorder
		c    echo.Context
		h    echo.HandlerFunc
		mw   echo.MiddlewareFunc
		err  error
		cases = []struct {
			name       string
			adminRole  string
			allowed    []model.AdminRole
			wantStatus int
		}{
			{
				name:       "super admin bypasses review only",
				adminRole:  string(model.AdminRoleSuperAdmin),
				allowed:    []model.AdminRole{model.AdminRoleReviewAdmin},
				wantStatus: http.StatusOK,
			},
			{
				name:       "review admin allowed",
				adminRole:  string(model.AdminRoleReviewAdmin),
				allowed:    []model.AdminRole{model.AdminRoleReviewAdmin},
				wantStatus: http.StatusOK,
			},
			{
				name:       "support admin denied for review",
				adminRole:  string(model.AdminRoleSupportAdmin),
				allowed:    []model.AdminRole{model.AdminRoleReviewAdmin},
				wantStatus: http.StatusForbidden,
			},
			{
				name:       "missing admin role",
				adminRole:  "",
				allowed:    []model.AdminRole{model.AdminRoleSupportAdmin},
				wantStatus: http.StatusForbidden,
			},
		}
		i int
		tc struct {
			name       string
			adminRole  string
			allowed    []model.AdminRole
			wantStatus int
		}
	)

	e = echo.New()
	h = func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	}
	for i = range cases {
		tc = cases[i]
		t.Run(tc.name, func(t *testing.T) {
			req = httptest.NewRequest(http.MethodGet, "/", nil)
			rec = httptest.NewRecorder()
			c = e.NewContext(req, rec)
			c.Set(CtxUserRole, string(model.UserRoleAdmin))
			if tc.adminRole != "" {
				c.Set(CtxAdminRole, tc.adminRole)
			}
			mw = RequireAdminRole(tc.allowed...)
			err = mw(h)(c)
			if tc.wantStatus == http.StatusOK {
				if err != nil {
					t.Fatalf("expected ok, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error")
			}
			he, ok := err.(*echo.HTTPError)
			if !ok || he.Code != tc.wantStatus {
				t.Fatalf("status=%v err=%v", he, err)
			}
		})
	}
}

func TestRequireAdminRoleNonAdminUser(t *testing.T) {
	var (
		e   *echo.Echo
		req *http.Request
		rec *httptest.ResponseRecorder
		c   echo.Context
		err error
	)

	e = echo.New()
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.Set(CtxUserRole, string(model.UserRoleBuyer))
	c.Set(CtxAdminRole, string(model.AdminRoleSuperAdmin))
	err = RequireAdminRole(model.AdminRoleSuperAdmin)(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})(c)
	if err == nil {
		t.Fatal("expected forbidden for non-admin user")
	}
}
