package controller

import (
	"net/url"
	"strings"
	"time"

	"crowdfund-app/usecase"
)

type registerResponse struct {
	User    UserWire `json:"user"`
	Message string   `json:"message"`
}

type login2FAResponse struct {
	Requires2FA bool     `json:"requires_2fa"`
	User        UserWire `json:"user"`
}

type authTokenResponse struct {
	AuthToken string   `json:"auth_token"`
	User      UserWire `json:"user"`
}

type verify2FAResponse struct {
	Enabled     bool     `json:"enabled"`
	BackupCodes []string `json:"backup_codes"`
}

type sellerCreateJSON struct {
	User         UserWire      `json:"user"`
	Registration sellerRegJSON `json:"registration"`
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

type updateProfileJSON struct {
	Name      *string `json:"name"`
	AvatarURL *string `json:"avatar_url"`
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
	if v.LinkCode != "" {
		q.Set("link_code", v.LinkCode)
	}
	if v.AccountEmail != "" {
		q.Set("account_email", v.AccountEmail)
	}
	if v.GoogleEmail != "" {
		q.Set("google_email", v.GoogleEmail)
	}
	if len(q) == 0 {
		return base
	}
	return base + "?" + q.Encode()
}
