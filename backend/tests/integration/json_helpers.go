//go:build integration

package integration

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

type apiCodeBody struct {
	Code string `json:"code"`
}

type favoriteStatusBody struct {
	IsFavorited bool `json:"is_favorited"`
}

type favoritesListBody struct {
	Favorites []json.RawMessage `json:"favorites"`
}

type sellerStatsBody struct {
	TotalProjects   int64 `json:"total_projects"`
	TotalSupporters int64 `json:"total_supporters"`
	TotalRevenue    int64 `json:"total_revenue"`
}

type sellerSupportersBody struct {
	Supporters []json.RawMessage `json:"supporters"`
}

type sellerProjectsAnalysisBody struct {
	Projects []json.RawMessage `json:"projects"`
}

type userProfileBody struct {
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

type reportBody struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type mediaUploadBody struct {
	URL string `json:"url"`
}

type purchaseStatusBody struct {
	Status string `json:"status"`
}

type securityEventsBody struct {
	LoginFailedCount int64 `json:"login_failed_count"`
	RateLimitedCount int64 `json:"rate_limited_count"`
}

type stripeWebhookObject struct {
	ID                 string            `json:"id"`
	Metadata           map[string]string `json:"metadata"`
	Status             string            `json:"status,omitempty"`
	PaymentStatus      string            `json:"payment_status,omitempty"`
	PaymentMethodTypes []string          `json:"payment_method_types,omitempty"`
}

type stripeWebhookData struct {
	Object json.RawMessage `json:"object"`
}

type stripeWebhookPayload struct {
	ID   string              `json:"id"`
	Type string              `json:"type"`
	Data stripeWebhookData   `json:"data"`
}

func bodyCode(rec *httptest.ResponseRecorder) string {
	var m apiCodeBody
	raw := strings.TrimSpace(rec.Body.String())
	if raw == "" || json.Unmarshal([]byte(raw), &m) != nil {
		return ""
	}
	return m.Code
}

func jsonStringField(t *testing.T, rec *httptest.ResponseRecorder, key string) string {
	var (
		fields map[string]json.RawMessage
		raw    json.RawMessage
		s      string
		ok     bool
	)
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), &fields); err != nil {
		t.Fatalf("json: %v body=%s", err, rec.Body.String())
	}
	raw, ok = fields[key]
	if !ok {
		t.Fatalf("missing %q in %s", key, rec.Body.String())
	}
	if err := json.Unmarshal(raw, &s); err != nil || s == "" {
		t.Fatalf("missing %q in %s", key, rec.Body.String())
	}
	return s
}

func buildStripeEventPayload(eventID, eventType, objectID string, meta map[string]string, konbini bool) ([]byte, error) {
	var (
		obj    stripeWebhookObject
		objRaw []byte
		err    error
	)
	obj = stripeWebhookObject{
		ID:       objectID,
		Metadata: meta,
	}
	switch eventType {
	case "payment_intent.succeeded":
		obj.Status = "succeeded"
	case "checkout.session.async_payment_succeeded", "checkout.session.completed":
		obj.PaymentStatus = "paid"
		if konbini {
			obj.PaymentMethodTypes = []string{"konbini"}
		}
	case "checkout.session.expired":
		if konbini {
			obj.PaymentMethodTypes = []string{"konbini"}
		}
	}
	objRaw, err = json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	return json.Marshal(stripeWebhookPayload{
		ID:   eventID,
		Type: eventType,
		Data: stripeWebhookData{Object: objRaw},
	})
}
