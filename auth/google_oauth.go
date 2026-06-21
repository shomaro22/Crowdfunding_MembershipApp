package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type GoogleProfile struct {
	Sub   string
	Email string
	Name  string
}

type GoogleOAuth struct {
	Config      oauth2.Config
	Cache       OAuthStateStore
	AppPublicURL string
}

type OAuthStateStore interface {
	SetState(ctx context.Context, state, from string) error
	PopState(ctx context.Context, state string) (from string, ok bool, err error)
}

func GoogleFromEnv(store OAuthStateStore) (*GoogleOAuth, error) {
	clientID := strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID"))
	clientSecret := strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_SECRET"))
	redirectURL := strings.TrimSpace(os.Getenv("GOOGLE_REDIRECT_URI"))
	if redirectURL == "" {
		apiBase := strings.TrimSpace(os.Getenv("API_PUBLIC_URL"))
		if apiBase == "" {
			port := strings.TrimSpace(os.Getenv("PORT"))
			if port == "" {
				port = "8080"
			}
			apiBase = "http://localhost:" + port
		}
		redirectURL = strings.TrimRight(apiBase, "/") + "/auth/google/callback"
	}
	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET required")
	}
	appURL := strings.TrimSpace(os.Getenv("PUBLIC_APP_URL"))
	if appURL == "" {
		appURL = "http://localhost:5173"
	}
	return &GoogleOAuth{
		Config: oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{"openid", "email", "profile"},
			Endpoint:     google.Endpoint,
		},
		Cache:        store,
		AppPublicURL: strings.TrimRight(appURL, "/"),
	}, nil
}

func (g *GoogleOAuth) AuthURL(from string) (state, url string, err error) {
	state = uuid.NewString()
	if g.Cache != nil {
		if err := g.Cache.SetState(context.Background(), state, from); err != nil {
			return "", "", err
		}
	}
	return state, g.Config.AuthCodeURL(state, oauth2.AccessTypeOnline), nil
}

func (g *GoogleOAuth) Exchange(ctx context.Context, code string) (*GoogleProfile, error) {
	tok, err := g.Config.Exchange(ctx, code)
	if err != nil {
		return nil, err
	}
	client := g.Config.Client(ctx, tok)
	resp, err := client.Get("https://openidconnect.googleapis.com/v1/userinfo")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("google userinfo: %s", strings.TrimSpace(string(body)))
	}
	var raw struct {
		Sub   string `json:"sub"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	if raw.Sub == "" || raw.Email == "" {
		return nil, fmt.Errorf("google profile incomplete")
	}
	return &GoogleProfile{Sub: raw.Sub, Email: strings.ToLower(strings.TrimSpace(raw.Email)), Name: raw.Name}, nil
}

func (g *GoogleOAuth) CallbackRedirect(from, query string) string {
	base := g.AppPublicURL + "/auth/callback"
	if from != "" {
		base += "?from=" + from
	}
	if query == "" {
		return base
	}
	sep := "?"
	if strings.Contains(base, "?") {
		sep = "&"
	}
	return base + sep + query
}
