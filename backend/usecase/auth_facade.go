package usecase

import (
	"context"

	"github.com/google/uuid"
)

// authFacade は 5 分割された認証ユースケースを IAuthUsecase として公開する。
type authFacade struct {
	session      *SessionAuthUseCase
	registration *RegistrationUseCase
	password     *PasswordUseCase
	profile      *ProfileUseCase
	oauth        *OAuthUseCase
	twoFactor    *TwoFactorUseCase
}

func (f *authFacade) RegisterBuyer(ctx context.Context, email, password string, name *string, meta RequestMeta) (UserView, error) {
	return f.registration.RegisterBuyer(ctx, email, password, name, meta)
}

func (f *authFacade) RegisterSeller(ctx context.Context, in SellerRegIn, meta RequestMeta) (*SellerRegOut, error) {
	return f.registration.RegisterSeller(ctx, in, meta)
}

func (f *authFacade) EmailFree(ctx context.Context, email string) (bool, error) {
	return f.registration.EmailFree(ctx, email)
}

func (f *authFacade) Login(ctx context.Context, email, password string, meta RequestMeta) (*LoginResult, error) {
	return f.session.Login(ctx, email, password, meta)
}

func (f *authFacade) Refresh(ctx context.Context, refreshTokenPlain string, meta RequestMeta) (string, string, error) {
	return f.session.Refresh(ctx, refreshTokenPlain, meta)
}

func (f *authFacade) Logout(ctx context.Context, refreshTokenPlain string, meta RequestMeta) error {
	return f.session.Logout(ctx, refreshTokenPlain, meta)
}

func (f *authFacade) LogoutAllDevices(ctx context.Context, userID uuid.UUID, meta RequestMeta) error {
	return f.session.LogoutAllDevices(ctx, userID, meta)
}

func (f *authFacade) ParseToken(raw string) (AuthClaims, error) {
	return f.session.ParseToken(raw)
}

func (f *authFacade) ValidateSession(ctx context.Context, claims AuthClaims) (SessionInfo, error) {
	return f.session.ValidateSession(ctx, claims)
}

func (f *authFacade) ForgotPassword(ctx context.Context, email string) error {
	return f.password.ForgotPassword(ctx, email)
}

func (f *authFacade) ResetPassword(ctx context.Context, plainToken, newPassword string) error {
	return f.password.ResetPassword(ctx, plainToken, newPassword)
}

func (f *authFacade) ChangePassword(ctx context.Context, userID uuid.UUID, current, newPassword, totpCode string) error {
	return f.password.ChangePassword(ctx, userID, current, newPassword, totpCode)
}

func (f *authFacade) GetUser(ctx context.Context, userID uuid.UUID) (UserView, error) {
	return f.profile.GetUser(ctx, userID)
}

func (f *authFacade) UpdateProfile(ctx context.Context, userID uuid.UUID, in UpdateProfileIn) (UserView, error) {
	return f.profile.UpdateProfile(ctx, userID, in)
}

func (f *authFacade) Setup2FA(ctx context.Context, userID uuid.UUID) (*TOTPSetupResult, error) {
	return f.twoFactor.Setup2FA(ctx, userID)
}

func (f *authFacade) Verify2FASetup(ctx context.Context, userID uuid.UUID, code string) (*TOTPVerifyResult, error) {
	return f.twoFactor.Verify2FASetup(ctx, userID, code)
}

func (f *authFacade) Challenge2FA(ctx context.Context, pendingToken, code string, meta RequestMeta) (*LoginResult, error) {
	return f.twoFactor.Challenge2FA(ctx, pendingToken, code, meta)
}

func (f *authFacade) CreatePending2FAToken(ctx context.Context, userID uuid.UUID) (string, error) {
	return f.twoFactor.CreatePending2FAToken(ctx, userID)
}

func (f *authFacade) GoogleAuthURL(from string, meta RequestMeta) (string, error) {
	return f.oauth.GoogleAuthURL(from, meta)
}

func (f *authFacade) GoogleCallback(ctx context.Context, code, state string, meta RequestMeta) (*OAuthCBOut, error) {
	return f.oauth.GoogleCallback(ctx, code, state, meta)
}

func (f *authFacade) ExchangeOAuthCode(ctx context.Context, code string, meta RequestMeta) (*LoginResult, error) {
	return f.oauth.ExchangeOAuthCode(ctx, code, meta)
}

func (f *authFacade) ConfirmOAuthLink(ctx context.Context, linkCode string, meta RequestMeta) (*LoginResult, error) {
	return f.oauth.ConfirmOAuthLink(ctx, linkCode, meta)
}

func (f *authFacade) OAuthEnabled() bool {
	return f.oauth.OAuthEnabled()
}

func newAuthDeps(cfg AuthConfig) *authDeps {
	return &authDeps{
		Users:         cfg.Users,
		SellerRegs:    cfg.SellerRegs,
		RefreshTokens: cfg.RefreshTokens,
		ResetToks:     cfg.ResetToks,
		Activity:      cfg.Activity,
		OAuth:         cfg.OAuth,
		OAuthCache:    cfg.OAuthCache,
		TOTPBackup:    cfg.TOTPBackup,
		Tokens:        NewTokenService([]byte(cfg.JWTSecret), DefaultAuthTTL),
		Mailer:        cfg.Mailer,
		KYCStore:      cfg.KYCStore,
		TOTPKey:       cfg.TOTPEncryptionKey,
		TOTPIssuer:    cfg.TOTPIssuer,
		pendingSecret: cfg.Pending2FAKey,
		Tx:            cfg.Tx,
		AppPublicURL:  cfg.AppPublicURL,
	}
}
