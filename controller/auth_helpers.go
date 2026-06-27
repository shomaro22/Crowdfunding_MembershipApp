package controller

import (
	"net/http"
	"net/url"
	"strings"

	"crowdfund-app/auth"
	"crowdfund-app/middleware"
	"crowdfund-app/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func CurrentUserID(c echo.Context) (uuid.UUID, error) {
	var (
		uid uuid.UUID
		ok  bool
	)
	uid, ok = middleware.AuthUserID(c)
	if !ok {
		return uuid.Nil, echo.NewHTTPError(http.StatusUnauthorized, apiError("unauthorized", "UNAUTHORIZED"))
	}
	return uid, nil
}

func requestMeta(c echo.Context) usecase.RequestMeta {
	var reqID string
	reqID, _ = c.Get(middleware.CtxRequestID).(string)
	return usecase.RequestMeta{
		IPAddress: c.RealIP(),
		UserAgent: c.Request().UserAgent(),
		RequestID: reqID,
	}
}

type registerResponse struct {
	User    usecase.UserWire `json:"user"`
	Message string           `json:"message"`
}

type login2FAResponse struct {
	Requires2FA bool             `json:"requires_2fa"`
	User        usecase.UserWire `json:"user"`
}

type authTokenResponse struct {
	AuthToken string           `json:"auth_token"`
	User      usecase.UserWire `json:"user"`
}

type verify2FAResponse struct {
	Enabled     bool     `json:"enabled"`
	BackupCodes []string `json:"backup_codes"`
}

type sellerCreateJSON struct {
	User         usecase.UserWire      `json:"user"`
	Registration sellerRegJSON         `json:"registration"`
}

type usersPageJSON struct {
	Users []usecase.UserWire `json:"users"`
	Total int64              `json:"total"`
}

type sellerRegsPageJSON struct {
	Registrations []sellerRegJSON `json:"registrations"`
	Total         int64           `json:"total"`
}

type sellerRegViewJSON struct {
	sellerRegJSON
	User     usecase.UserWire `json:"user"`
	FrontURL *string          `json:"identity_document_front_url,omitempty"`
	BackURL  *string          `json:"identity_document_back_url,omitempty"`
}

func sellerRegViewToJSON(apiBase string, d *usecase.SellerRegView) sellerRegViewJSON {
	var (
		out sellerRegViewJSON
		url string
	)
	out = sellerRegViewJSON{
		sellerRegJSON: sellerRegToJSON(d.Registration),
		User:          usecase.UserWireFrom(d.User),
	}
	if d.FrontAccessToken != nil {
		url = auth.KYCAccessURL(apiBase, *d.FrontAccessToken)
		out.FrontURL = &url
	}
	if d.BackAccessToken != nil {
		url = auth.KYCAccessURL(apiBase, *d.BackAccessToken)
		out.BackURL = &url
	}
	return out
}

func oauthCallbackURL(appPublicURL string, v usecase.OAuthRedirectView) string {
	var (
		base string
		q    url.Values
	)
	base = strings.TrimRight(strings.TrimSpace(appPublicURL), "/") + "/auth/callback"
	if v.Error != "" {
		return base + "?error=" + url.QueryEscape(v.Error)
	}
	q = url.Values{}
	if v.From != "" {
		q.Set("from", v.From)
	}
	if v.Code != "" {
		q.Set("code", v.Code)
	}
	if len(q) == 0 {
		return base
	}
	return base + "?" + q.Encode()
}
