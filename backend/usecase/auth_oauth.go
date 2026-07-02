package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"crowdfund-app/model"
	"crowdfund-app/repository"

	"github.com/google/uuid"
)

const oauthCallbackCodeTTL = 2 * time.Minute

type OAuthRedirectView struct {
	Code         string
	LinkCode     string
	AccountEmail string
	GoogleEmail  string
	From         string
	Error        string
}

type OAuthCBOut struct {
	RefreshToken string
	Redirect     OAuthRedirectView
}

type googleResolveResult struct {
	User             model.User
	NeedsLinkConfirm bool
	Profile          *GoogleProfile
}

func (o *OAuthUseCase) OAuthEnabled() bool {
	return o != nil && o.deps.Google != nil
}

func (o *OAuthUseCase) GoogleAuthURL(from string, meta RequestMeta) (string, error) {
	var (
		authURL string
		err     error
	)

	if o.deps.Google == nil {
		return "", ErrOAuthUnavailable
	}
	_, authURL, err = o.deps.Google.AuthURL(from, meta)
	return authURL, err
}

func (o *OAuthUseCase) GoogleCallback(ctx context.Context, code, state string, meta RequestMeta) (*OAuthCBOut, error) {
	var (
		from      string
		ok        bool
		err       error
		stateData repository.OAuthStateData
		profile   *GoogleProfile
		resolved  googleResolveResult
		tokens    *SessionTokens
		cbCode    string
		linkCode  string
	)

	if o.deps.Google == nil || o.deps.OAuthCache == nil || o.deps.OAuth == nil {
		return nil, ErrOAuthUnavailable
	}
	stateData, ok, err = o.deps.OAuthCache.PopOAuthState(ctx, state)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrInvalidToken
	}
	from = stateData.From
	profile, err = o.deps.Google.Exchange(ctx, code, stateData)
	if err != nil {
		return nil, err
	}
	resolved, err = o.resolveGoogleUser(ctx, profile)
	if err != nil {
		return nil, err
	}
	if resolved.User.Role != model.UserRoleBuyer {
		return nil, ErrOAuthForbidden
	}
	if !resolved.User.CanLogin() {
		return nil, ErrInactiveUser
	}
	if resolved.NeedsLinkConfirm {
		linkCode, err = OpaqueHex()
		if err != nil {
			return nil, err
		}
		err = o.deps.OAuthCache.SetLinkCode(ctx, linkCode, repository.OAuthLinkPending{
			UserID:      resolved.User.ID,
			ProviderSub: profile.Sub,
			GoogleEmail: profile.Email,
			GoogleName:  profile.Name,
		}, oauthCallbackCodeTTL)
		if err != nil {
			return nil, err
		}
		return &OAuthCBOut{
			Redirect: OAuthRedirectView{
				LinkCode:     linkCode,
				AccountEmail: resolved.User.Email,
				GoogleEmail:  profile.Email,
				From:         from,
			},
		}, nil
	}
	tokens, err = o.deps.issueSession(ctx, resolved.User, twoFAVerified(resolved.User))
	if err != nil {
		return nil, err
	}
	cbCode, err = OpaqueHex()
	if err != nil {
		return nil, err
	}
	if err = o.deps.OAuthCache.SetCallbackCode(ctx, cbCode, resolved.User.ID, oauthCallbackCodeTTL); err != nil {
		return nil, err
	}
	o.deps.logActivity(ctx, &resolved.User.ID, model.UserLogin, "user", resolved.User.ID.String(), meta, model.ProviderMeta("google"))
	return &OAuthCBOut{
		RefreshToken: tokens.RefreshToken,
		Redirect: OAuthRedirectView{
			Code: cbCode,
			From: from,
		},
	}, nil
}

func (o *OAuthUseCase) ExchangeOAuthCode(ctx context.Context, code string, meta RequestMeta) (*LoginResult, error) {
	var (
		userID    uuid.UUID
		ok        bool
		err       error
		u         model.User
		authToken string
	)

	if o.deps.OAuthCache == nil {
		return nil, ErrOAuthUnavailable
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, ErrInvalidToken
	}
	userID, ok, err = o.deps.OAuthCache.PopCallback(ctx, code)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrInvalidToken
	}
	u, err = o.deps.Users.GetID(ctx, userID)
	if err != nil {
		return nil, mapRepoNotFoundAs(err, ErrInvalidToken)
	}
	if !u.CanLogin() {
		return nil, ErrInvalidToken
	}
	authToken, err = o.deps.Tokens.Issue(AuthClaims{
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

func (o *OAuthUseCase) ConfirmOAuthLink(ctx context.Context, linkCode string, meta RequestMeta) (*LoginResult, error) {
	var (
		pending repository.OAuthLinkPending
		ok      bool
		err     error
		u       model.User
		now     time.Time
		link    *model.OAuthAccount
		tokens  *SessionTokens
	)

	if o.deps.OAuthCache == nil || o.deps.OAuth == nil {
		return nil, ErrOAuthUnavailable
	}
	linkCode = strings.TrimSpace(linkCode)
	if linkCode == "" {
		return nil, ErrInvalidToken
	}
	pending, ok, err = o.deps.OAuthCache.PopLinkCode(ctx, linkCode)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrInvalidToken
	}
	u, err = o.deps.Users.GetID(ctx, pending.UserID)
	if err != nil {
		return nil, mapRepoNotFoundAs(err, ErrInvalidToken)
	}
	if u.Role != model.UserRoleBuyer || !u.CanLogin() {
		return nil, ErrOAuthForbidden
	}
	now = time.Now().UTC()
	link = &model.OAuthAccount{
		ID:          uuid.New(),
		UserID:      u.ID,
		Provider:    model.OAuthProviderGoogle,
		ProviderSub: pending.ProviderSub,
		Email:       pending.GoogleEmail,
		CreatedAt:   now,
	}
	if err = o.deps.OAuth.Create(ctx, link); err != nil {
		return nil, err
	}
	tokens, err = o.deps.issueSession(ctx, u, twoFAVerified(u))
	if err != nil {
		return nil, err
	}
	o.deps.logActivity(ctx, &u.ID, model.UserLogin, "user", u.ID.String(), meta, model.ProviderMeta("google"))
	return &LoginResult{
		User:         UserViewFrom(u),
		AuthToken:    tokens.AuthToken,
		RefreshToken: tokens.RefreshToken,
	}, nil
}

func (o *OAuthUseCase) resolveGoogleUser(ctx context.Context, profile *GoogleProfile) (googleResolveResult, error) {
	var (
		acc      model.OAuthAccount
		err      error
		existing model.User
		now      time.Time
		link     *model.OAuthAccount
		pwPlain  string
		pwHash   string
		name     *string
		n        string
		u        model.User
		hasAcc   bool
		out      googleResolveResult
	)

	acc, err = o.deps.OAuth.GetSub(ctx, model.OAuthProviderGoogle, profile.Sub)
	if isRepoNotFound(err) {
		hasAcc = false
	} else if err != nil {
		return googleResolveResult{}, err
	} else {
		hasAcc = true
	}
	if hasAcc {
		u, err = o.deps.Users.GetID(ctx, acc.UserID)
		if err != nil {
			return googleResolveResult{}, mapRepoNotFound(err)
		}
		out.User = u
		return out, nil
	}
	existing, err = o.deps.Users.GetEmail(ctx, profile.Email)
	if isRepoNotFound(err) {
		existing = model.User{}
	} else if err != nil {
		return googleResolveResult{}, err
	}
	now = time.Now().UTC()
	if existing.ID != uuid.Nil {
		if existing.Role != model.UserRoleBuyer {
			return googleResolveResult{}, ErrOAuthForbidden
		}
		if existing.AccountStatus == model.AccountStatusPending {
			existing.AccountStatus = model.AccountStatusActive
			existing.UpdatedAt = now
			if err = o.deps.Users.Update(ctx, &existing); err != nil {
				return googleResolveResult{}, err
			}
		}
		out.User = existing
		out.NeedsLinkConfirm = true
		out.Profile = profile
		return out, nil
	}
	pwPlain, err = OpaqueHex()
	if err != nil {
		return googleResolveResult{}, err
	}
	pwHash, err = HashPassword(pwPlain)
	if err != nil {
		return googleResolveResult{}, err
	}
	if profile.Name != "" {
		n = profile.Name
		name = &n
	}
	u = model.User{
		ID:            uuid.New(),
		Email:         profile.Email,
		Name:          name,
		PasswordHash:  pwHash,
		Role:          model.UserRoleBuyer,
		AccountStatus: model.AccountStatusActive,
		SellerStatus:  model.SellerStatusNotApplicable,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err = o.deps.Users.Create(ctx, &u); err != nil {
		return googleResolveResult{}, err
	}
	link = &model.OAuthAccount{
		ID:          uuid.New(),
		UserID:      u.ID,
		Provider:    model.OAuthProviderGoogle,
		ProviderSub: profile.Sub,
		Email:       profile.Email,
		CreatedAt:   now,
	}
	if err = o.deps.OAuth.Create(ctx, link); err != nil {
		return googleResolveResult{}, fmt.Errorf("oauth link: %w", err)
	}
	out.User = u
	return out, nil
}
