package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"crowdfund-app/auth"
	"crowdfund-app/entity"

	"github.com/google/uuid"
)

const oauthCallbackCodeTTL = 2 * time.Minute

type OAuthCBOut struct {
	RefreshToken string
	CallbackCode string
	From         string
}

func (a *AuthUseCase) GoogleAuthURL(from string) (string, error) {
	var (
		authURL string
		err     error
	)

	if a.Google == nil {
		return "", ErrOAuthUnavailable
	}
	_, authURL, err = a.Google.AuthURL(from)
	return authURL, err
}

func (a *AuthUseCase) GoogleCallback(ctx context.Context, code, state string, meta RequestMeta) (*OAuthCBOut, error) {
	var (
		from    string
		ok      bool
		err     error
		profile *auth.GoogleProfile
		u       *entity.User
		tokens  *SessionTokens
		cbCode  string
	)

	if a.Google == nil || a.OAuthCache == nil || a.OAuth == nil {
		return nil, ErrOAuthUnavailable
	}
	from, ok, err = a.OAuthCache.PopState(ctx, state)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrInvalidToken
	}
	profile, err = a.Google.Exchange(ctx, code)
	if err != nil {
		return nil, err
	}
	u, err = a.resolveGoogleUser(ctx, profile)
	if err != nil {
		return nil, err
	}
	if u.Role != entity.UserRoleBuyer {
		return nil, ErrOAuthForbidden
	}
	if !u.CanLogin() {
		return nil, ErrInactiveUser
	}
	tokens, err = a.issueSession(ctx, u, twoFAVerified(u))
	if err != nil {
		return nil, err
	}
	cbCode, err = auth.OpaqueHex()
	if err != nil {
		return nil, err
	}
	if err = a.OAuthCache.SetCallbackCode(ctx, cbCode, u.ID, oauthCallbackCodeTTL); err != nil {
		return nil, err
	}
	a.logActivity(ctx, &u.ID, entity.UserLogin, "user", u.ID.String(), meta, entity.ProviderMeta("google"))
	return &OAuthCBOut{
		RefreshToken: tokens.RefreshToken,
		CallbackCode: cbCode,
		From:         from,
	}, nil
}

func (a *AuthUseCase) ExchangeOAuthCode(ctx context.Context, code string, meta RequestMeta) (*LoginResult, error) {
	var (
		userID    uuid.UUID
		ok        bool
		err       error
		u         *entity.User
		authToken string
	)

	if a.OAuthCache == nil {
		return nil, ErrOAuthUnavailable
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, ErrInvalidToken
	}
	userID, ok, err = a.OAuthCache.PopCallback(ctx, code)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrInvalidToken
	}
	u, err = a.Users.FindID(ctx, userID)
	if err != nil {
		return nil, mapRepoNotFoundAs(err, ErrInvalidToken)
	}
	if !u.CanLogin() {
		return nil, ErrInvalidToken
	}
	authToken, err = a.Tokens.Issue(auth.AuthClaims{
		UserID:        u.ID,
		Role:          u.Role,
		TwoFAVerified: twoFAVerified(u),
		AdminRole:     u.AdminRole,
		TokenVersion:  u.TokenVersion,
	})
	if err != nil {
		return nil, err
	}
	return &LoginResult{User: UserViewFrom(u), AuthToken: authToken}, nil
}

func (a *AuthUseCase) resolveGoogleUser(ctx context.Context, profile *auth.GoogleProfile) (*entity.User, error) {
	var (
		acc      *entity.OAuthAccount
		err      error
		existing *entity.User
		now      time.Time
		link     *entity.OAuthAccount
		pwPlain  string
		pwHash   string
		name     *string
		n        string
		u        *entity.User
	)

	acc, err = a.OAuth.FindSub(ctx, entity.OAuthProviderGoogle, profile.Sub)
	if isRepoNotFound(err) {
		acc = nil
	} else if err != nil {
		return nil, err
	}
	if acc != nil {
		u, err = a.Users.FindID(ctx, acc.UserID)
		if err != nil {
			return nil, mapRepoNotFound(err)
		}
		return u, nil
	}
	existing, err = a.Users.FindEmail(ctx, profile.Email)
	if isRepoNotFound(err) {
		existing = nil
	} else if err != nil {
		return nil, err
	}
	now = time.Now().UTC()
	if existing != nil {
		if existing.Role != entity.UserRoleBuyer {
			return nil, ErrOAuthForbidden
		}
		link = &entity.OAuthAccount{
			ID:          uuid.New(),
			UserID:      existing.ID,
			Provider:    entity.OAuthProviderGoogle,
			ProviderSub: profile.Sub,
			Email:       profile.Email,
			CreatedAt:   now,
		}
		if err = a.OAuth.Create(ctx, link); err != nil {
			return nil, err
		}
		return existing, nil
	}
	pwPlain, err = auth.OpaqueHex()
	if err != nil {
		return nil, err
	}
	pwHash, err = auth.HashPassword(pwPlain)
	if err != nil {
		return nil, err
	}
	if profile.Name != "" {
		n = profile.Name
		name = &n
	}
	u = &entity.User{
		ID:            uuid.New(),
		Email:         profile.Email,
		Name:          name,
		PasswordHash:  pwHash,
		Role:          entity.UserRoleBuyer,
		AccountStatus: entity.AccountStatusPending,
		SellerStatus:  entity.SellerStatusNotApplicable,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err = a.Users.Create(ctx, u); err != nil {
		return nil, err
	}
	link = &entity.OAuthAccount{
		ID:          uuid.New(),
		UserID:      u.ID,
		Provider:    entity.OAuthProviderGoogle,
		ProviderSub: profile.Sub,
		Email:       profile.Email,
		CreatedAt:   now,
	}
	if err = a.OAuth.Create(ctx, link); err != nil {
		return nil, fmt.Errorf("oauth link: %w", err)
	}
	return u, nil
}
