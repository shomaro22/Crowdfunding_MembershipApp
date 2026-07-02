package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"crowdfund-app/repository"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var oauthHTTPClient = &http.Client{Timeout: 15 * time.Second}

type GoogleProfile struct {
	Sub   string
	Email string
	Name  string
}

type GoogleOAuth struct {
	Config       oauth2.Config
	Cache        OAuthStateStore
	AppPublicURL string
}

type OAuthStateStore interface {
	SetOAuthState(ctx context.Context, state string, data repository.OAuthStateData) error
	PopOAuthState(ctx context.Context, state string) (repository.OAuthStateData, bool, error)
}

type oauthStateBridge struct {
	cache repository.IOAuthCache
	ttl   time.Duration
}

func NewOAuthStateBridge(cache repository.IOAuthCache) OAuthStateStore {
	return &oauthStateBridge{cache: cache, ttl: 10 * time.Minute}
}

func (b *oauthStateBridge) SetOAuthState(ctx context.Context, state string, data repository.OAuthStateData) error {
	if b.cache == nil {
		return ErrInvalidToken
	}
	return b.cache.SetOAuthState(ctx, state, data, b.ttl)
}

func (b *oauthStateBridge) PopOAuthState(ctx context.Context, state string) (repository.OAuthStateData, bool, error) {
	if b.cache == nil {
		return repository.OAuthStateData{}, false, ErrInvalidToken
	}
	return b.cache.PopOAuthState(ctx, state)
}

func GoogleFromEnv(store OAuthStateStore) (*GoogleOAuth, error) {
	var (
		clientID     string
		clientSecret string
		redirectURL  string
		apiBase      string
		port         string
		appURL       string
	)
	clientID = strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_ID"))
	clientSecret = strings.TrimSpace(os.Getenv("GOOGLE_CLIENT_SECRET"))
	redirectURL = strings.TrimSpace(os.Getenv("GOOGLE_REDIRECT_URI"))
	if redirectURL == "" {
		apiBase = strings.TrimSpace(os.Getenv("API_PUBLIC_URL"))
		if apiBase == "" {
			port = strings.TrimSpace(os.Getenv("PORT"))
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
	appURL = strings.TrimSpace(os.Getenv("PUBLIC_APP_URL"))
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

func generatePKCE() (verifier string, challenge string, err error) {
	var (
		raw []byte
		sum [32]byte
	)
	raw = make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return "", "", err
	}
	verifier = base64.RawURLEncoding.EncodeToString(raw)
	sum = sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge, nil
}

func (g *GoogleOAuth) AuthURL(from string, meta RequestMeta) (state, url string, err error) {
	var (
		verifier  string
		challenge string
		nonce     string
		data      repository.OAuthStateData
	)
	state = uuid.NewString()
	nonce = uuid.NewString()
	verifier, challenge, err = generatePKCE()
	if err != nil {
		return "", "", err
	}
	data = repository.OAuthStateData{
		From:         from,
		CodeVerifier: verifier,
		IP:           meta.IPAddress,
		UserAgent:    meta.UserAgent,
		Nonce:        nonce,
	}
	if g.Cache != nil {
		err = g.Cache.SetOAuthState(context.Background(), state, data)
		if err != nil {
			return "", "", err
		}
	}
	return state, g.Config.AuthCodeURL(state,
		oauth2.AccessTypeOnline,
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	), nil
}

func (g *GoogleOAuth) Exchange(ctx context.Context, code string, stateData repository.OAuthStateData) (*GoogleProfile, error) {
	var (
		tok    *oauth2.Token
		err    error
		client *http.Client
		resp   *http.Response
		body   []byte
		raw    struct {
			Sub   string `json:"sub"`
			Email string `json:"email"`
			Name  string `json:"name"`
		}
	)
	ctx = context.WithValue(ctx, oauth2.HTTPClient, oauthHTTPClient)
	tok, err = g.Config.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", stateData.CodeVerifier))
	if err != nil {
		return nil, err
	}
	client = g.Config.Client(ctx, tok)
	resp, err = client.Get("https://openidconnect.googleapis.com/v1/userinfo")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ = io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("google userinfo: %s", strings.TrimSpace(string(body)))
	}
	if err = json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	if raw.Sub == "" || raw.Email == "" {
		return nil, fmt.Errorf("google profile incomplete")
	}
	return &GoogleProfile{
		Sub:   raw.Sub,
		Email: strings.ToLower(strings.TrimSpace(raw.Email)),
		Name:  raw.Name,
	}, nil
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
