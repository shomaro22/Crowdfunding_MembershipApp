package usecase

import (
	"context"
	"log/slog"
	"time"

	"crowdfund-app/model"
	"crowdfund-app/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// IAuthUsecase — controller/auth*.go と middleware が依存する認証ユースケース
type IAuthUsecase interface {
	RegisterBuyer(ctx context.Context, email, password string, name *string, meta RequestMeta) (UserView, error)
	RegisterSeller(ctx context.Context, in SellerRegIn, meta RequestMeta) (*SellerRegOut, error)
	Login(ctx context.Context, email, password string, meta RequestMeta) (*LoginResult, error)
	Refresh(ctx context.Context, refreshTokenPlain string, meta RequestMeta) (authToken, refreshToken string, err error)
	Logout(ctx context.Context, refreshTokenPlain string, meta RequestMeta) error
	LogoutAllDevices(ctx context.Context, userID uuid.UUID, meta RequestMeta) error
	ForgotPassword(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, plainToken, newPassword string) error
	ChangePassword(ctx context.Context, userID uuid.UUID, current, newPassword, totpCode string) error
	GetUser(ctx context.Context, userID uuid.UUID) (UserView, error)
	UpdateProfile(ctx context.Context, userID uuid.UUID, in UpdateProfileIn) (UserView, error)
	EmailFree(ctx context.Context, email string) (bool, error)
	Setup2FA(ctx context.Context, userID uuid.UUID) (*TOTPSetupResult, error)
	Verify2FASetup(ctx context.Context, userID uuid.UUID, code string) (*TOTPVerifyResult, error)
	Challenge2FA(ctx context.Context, pendingToken, code string, meta RequestMeta) (*LoginResult, error)
	CreatePending2FAToken(ctx context.Context, userID uuid.UUID) (string, error)
	GoogleAuthURL(from string, meta RequestMeta) (string, error)
	GoogleCallback(ctx context.Context, code, state string, meta RequestMeta) (*OAuthCBOut, error)
	ExchangeOAuthCode(ctx context.Context, code string, meta RequestMeta) (*LoginResult, error)
	ConfirmOAuthLink(ctx context.Context, linkCode string, meta RequestMeta) (*LoginResult, error)
	ParseToken(raw string) (AuthClaims, error)
	ValidateSession(ctx context.Context, claims AuthClaims) (SessionInfo, error)
	OAuthEnabled() bool
}

type AuthConfig struct {
	Users             repository.IUserRepo
	SellerRegs        repository.ISellerRegRepo
	RefreshTokens     repository.IRefreshTokenRepo
	ResetToks         repository.IResetRepo
	Activity          repository.IActivityRepo
	OAuth             repository.IOAuthRepo
	OAuthCache        repository.IOAuthCache
	TOTPBackup        repository.ITOTPBackupRepo
	KYCStore          repository.IKYCStore
	JWTSecret         string
	TOTPEncryptionKey []byte
	TOTPIssuer        string
	Pending2FAKey     []byte
	AppPublicURL      string
	Mailer            Mailer
	DB                *gorm.DB
}

func NewAuthUsecase(cfg AuthConfig) IAuthUsecase {
	var (
		deps   *authDeps
		google *GoogleOAuth
		err    error
	)

	deps = newAuthDeps(cfg)
	google, err = GoogleFromEnv(NewOAuthStateBridge(cfg.OAuthCache))
	if err == nil {
		deps.Google = google
	} else {
		slog.Info("google oauth disabled", "reason", err.Error())
	}
	return &authFacade{
		session:      &SessionAuthUseCase{deps: deps},
		registration: &RegistrationUseCase{deps: deps},
		password:     &PasswordUseCase{deps: deps},
		profile:      &ProfileUseCase{deps: deps},
		oauth:        &OAuthUseCase{deps: deps},
		twoFactor:    &TwoFactorUseCase{deps: deps},
	}
}

const (
	refreshTokenTTL = 7 * 24 * time.Hour
	resetTokenTTL   = time.Hour
)

type Mailer interface {
	SendPasswordReset(ctx context.Context, email, plainToken string) error
}

type SessionTokens struct {
	AuthToken    string
	RefreshToken string
}

type LoginResult struct {
	User         UserView
	AuthToken    string
	RefreshToken string
	Requires2FA  bool
}

type SessionInfo struct {
	Role      model.UserRole
	AdminRole *model.AdminRole
}
