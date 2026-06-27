package controller

import (
	"net/http"
	"net/url"
	"strings"
	"time"

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

func userToJSON(u usecase.UserView) userJSON {
	return userJSON{
		ID:        u.ID.String(),
		Email:     u.Email,
		Name:      u.Name,
		Role:      string(u.Role),
		IsActive:  u.IsActive(),
		AvatarURL: u.AvatarURL,
		CreatedAt: u.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: u.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

type userJSON struct {
	ID        string  `json:"id"`
	Email     string  `json:"email"`
	Name      *string `json:"name"`
	Role      string  `json:"role"`
	IsActive  bool    `json:"is_active"`
	AvatarURL *string `json:"avatar_url"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

func apiError(msg, code string) map[string]string {
	return map[string]string{"error": msg, "code": code}
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

type registerResponse struct {
	User    userJSON `json:"user"`
	Message string   `json:"message"`
}

type login2FAResponse struct {
	Requires2FA bool     `json:"requires_2fa"`
	User        userJSON `json:"user"`
}

type authTokenResponse struct {
	AuthToken string   `json:"auth_token"`
	User      userJSON `json:"user"`
}

type verify2FAResponse struct {
	Enabled     bool     `json:"enabled"`
	BackupCodes []string `json:"backup_codes"`
}

type sellerCreateJSON struct {
	User         userJSON      `json:"user"`
	Registration sellerRegJSON `json:"registration"`
}

type usersPageJSON struct {
	Users []userJSON `json:"users"`
	Total int64      `json:"total"`
}

type sellerRegsPageJSON struct {
	Registrations []sellerRegJSON `json:"registrations"`
	Total         int64           `json:"total"`
}

type sellerRegViewJSON struct {
	sellerRegJSON
	User     userJSON `json:"user"`
	FrontURL *string  `json:"identity_document_front_url,omitempty"`
	BackURL  *string  `json:"identity_document_back_url,omitempty"`
}

func sellerRegViewToJSON(apiBase string, d *usecase.SellerRegView) sellerRegViewJSON {
	var (
		out sellerRegViewJSON
		url string
	)
	out = sellerRegViewJSON{
		sellerRegJSON: sellerRegToJSON(d.Registration),
		User:          userToJSON(d.User),
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
