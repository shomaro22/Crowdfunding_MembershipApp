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

type registerResponse struct {
	User    UserWire `json:"user"`
	Message string           `json:"message"`
}

type login2FAResponse struct {
	Requires2FA bool             `json:"requires_2fa"`
	User        UserWire `json:"user"`
}

type authTokenResponse struct {
	AuthToken string           `json:"auth_token"`
	User      UserWire `json:"user"`
}

type verify2FAResponse struct {
	Enabled     bool     `json:"enabled"`
	BackupCodes []string `json:"backup_codes"`
}

type sellerCreateJSON struct {
	User         UserWire `json:"user"`
	Registration sellerRegJSON    `json:"registration"`
}

type sellerRegJSON struct {
	ID           string  `json:"id"`
	UserID       string  `json:"user_id"`
	FullName     string  `json:"full_name"`
	BirthDate    string  `json:"birth_date"`
	Address      string  `json:"address"`
	PhotoPresent bool    `json:"photo_present"`
	IDDocType    string  `json:"identity_document_type"`
	HasIDFront   bool    `json:"has_identity_document_front"`
	HasIDBack    bool    `json:"has_identity_document_back"`
	Status       string  `json:"status"`
	RejectReason *string `json:"rejection_reason"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

func sellerRegToJSON(r usecase.SellerRegItemView) sellerRegJSON {
	return sellerRegJSON{
		ID:           r.ID.String(),
		UserID:       r.UserID.String(),
		FullName:     r.FullName,
		BirthDate:    r.BirthDate.Format("2006-01-02"),
		Address:      r.Address,
		PhotoPresent: r.PhotoPresent,
		IDDocType:    "DRIVERS_LICENSE",
		HasIDFront:   r.HasIDFront,
		HasIDBack:    r.HasIDBack,
		Status:       r.Status,
		RejectReason: r.RejectReason,
		CreatedAt:    r.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:    r.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

type usersPageJSON struct {
	Users []UserWire `json:"users"`
	Total int64              `json:"total"`
}

type sellerRegsPageJSON struct {
	Registrations []sellerRegJSON `json:"registrations"`
	Total         int64           `json:"total"`
}

type sellerRegViewJSON struct {
	sellerRegJSON
	User     UserWire `json:"user"`
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
		User:          UserWireFrom(d.User),
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
