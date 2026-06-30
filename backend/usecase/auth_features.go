package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	
	"crowdfund-app/model"
	"crowdfund-app/repository"
	"crowdfund-app/validator"

	"github.com/google/uuid"
)

type SellerRegIn struct {
	Email        string
	Password     string
	FullName     string
	BirthDate    time.Time
	Address      string
	PhotoPresent bool
	Front        KYCUpload
	Back         KYCUpload
}

type SellerRegOut struct {
	User         UserView
	Registration SellerRegItemView
}

func (a *AuthUseCase) RegisterBuyer(ctx context.Context, email, password string, name *string, meta RequestMeta) (UserView, error) {
	var (
		normalized  string
		err         error
		exists      bool
		hash        string
		now         time.Time
		trimmedName *string
		n           string
		u           *model.User
	)

	normalized, err = validator.ValidateBuyerReg(email, password)
	if err != nil {
		return UserView{}, mapBuyerRegModelErr(err)
	}
	exists, err = a.Users.EmailExists(ctx, normalized)
	if err != nil {
		return UserView{}, err
	}
	if exists {
		return UserView{}, ErrEmailInUse
	}
	hash, err = HashPassword(password)
	if err != nil {
		return UserView{}, err
	}
	now = time.Now().UTC()
	if name != nil {
		n = strings.TrimSpace(*name)
		if n != "" {
			trimmedName = &n
		}
	}
	u = &model.User{
		ID:            uuid.New(),
		Email:         normalized,
		Name:          trimmedName,
		PasswordHash:  hash,
		Role:          model.UserRoleBuyer,
		AccountStatus: model.AccountStatusPending,
		SellerStatus:  model.SellerStatusNotApplicable,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err = a.Users.Create(ctx, u); err != nil {
		return UserView{}, err
	}
	a.logActivity(ctx, &u.ID, model.UserRegister, "user", u.ID.String(), meta, nil)
	return UserViewFrom(*u), nil
}

func (a *AuthUseCase) EmailFree(ctx context.Context, email string) (bool, error) {
	var (
		normalized string
		err        error
		exists     bool
	)

	normalized, err = validator.ValidateEmail(email)
	if err != nil {
		return false, ErrInvalidEmail
	}
	exists, err = a.Users.EmailExists(ctx, normalized)
	if err != nil {
		return false, err
	}
	return !exists, nil
}

func (a *AuthUseCase) RegisterSeller(ctx context.Context, in SellerRegIn, meta RequestMeta) (*SellerRegOut, error) {
	var (
		normalized string
		err        error
		exists     bool
		userID     uuid.UUID
		frontKey   string
		backKey    string
		hash       string
		now        time.Time
		u          *model.User
		reg        *model.SellerReg
	)

	if a.KYCStore == nil || a.SellerRegs == nil || a.DB == nil {
		return nil, ErrNotFound
	}

	normalized, err = validator.ValidateBuyerReg(in.Email, in.Password)
	if err != nil {
		return nil, mapBuyerRegModelErr(err)
	}
	if err = validator.ValidateSellerKYC(in.FullName, in.BirthDate, in.Address, in.PhotoPresent); err != nil {
		return nil, mapSellerKYCModelErr(err)
	}

	exists, err = a.Users.EmailExists(ctx, normalized)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrEmailInUse
	}

	userID = uuid.New()
	frontKey, err = a.KYCStore.Save(ctx, userID, "front", in.Front.toStoreImage())
	if err != nil {
		return nil, mapMediaStoreError(err)
	}
	backKey, err = a.KYCStore.Save(ctx, userID, "back", in.Back.toStoreImage())
	if err != nil {
		_ = a.KYCStore.Remove(ctx, frontKey)
		return nil, mapMediaStoreError(err)
	}

	hash, err = HashPassword(in.Password)
	if err != nil {
		_ = a.KYCStore.Remove(ctx, frontKey, backKey)
		return nil, err
	}

	now = time.Now().UTC()
	u = &model.User{
		ID:            userID,
		Email:         normalized,
		PasswordHash:  hash,
		Role:          model.UserRoleSeller,
		AccountStatus: model.AccountStatusPending,
		SellerStatus:  model.SellerStatusPending,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	reg = &model.SellerReg{
		ID:           uuid.New(),
		UserID:       userID,
		FullName:     strings.TrimSpace(in.FullName),
		BirthDate:    in.BirthDate,
		Address:      strings.TrimSpace(in.Address),
		PhotoPresent: in.PhotoPresent,
		IDFrontKey:   &frontKey,
		IDBackKey:    &backKey,
		Status:       model.RegPending,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	err = repository.WithTx(ctx, a.DB, func(txCtx context.Context) error {
		var txErr error

		if txErr = a.Users.Create(txCtx, u); txErr != nil {
			return txErr
		}
		return a.SellerRegs.Create(txCtx, reg)
	})
	if err != nil {
		_ = a.KYCStore.Remove(ctx, frontKey, backKey)
		return nil, err
	}

	a.logActivity(ctx, &u.ID, model.SellerRegister, "seller_registration", reg.ID.String(), meta, nil)
	return &SellerRegOut{User: UserViewFrom(*u), Registration: SellerRegItemFrom(*reg)}, nil
}

const oauthCallbackCodeTTL = 2 * time.Minute

type OAuthRedirectView struct {
	Code  string
	From  string
	Error string
}

type OAuthCBOut struct {
	RefreshToken string
	Redirect     OAuthRedirectView
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
		profile *GoogleProfile
		u       model.User
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
	if u.Role != model.UserRoleBuyer {
		return nil, ErrOAuthForbidden
	}
	if !u.CanLogin() {
		return nil, ErrInactiveUser
	}
	tokens, err = a.issueSession(ctx, u, twoFAVerified(u))
	if err != nil {
		return nil, err
	}
	cbCode, err = OpaqueHex()
	if err != nil {
		return nil, err
	}
	if err = a.OAuthCache.SetCallbackCode(ctx, cbCode, u.ID, oauthCallbackCodeTTL); err != nil {
		return nil, err
	}
	a.logActivity(ctx, &u.ID, model.UserLogin, "user", u.ID.String(), meta, model.ProviderMeta("google"))
	return &OAuthCBOut{
		RefreshToken: tokens.RefreshToken,
		Redirect: OAuthRedirectView{
			Code: cbCode,
			From: from,
		},
	}, nil
}

func (a *AuthUseCase) ExchangeOAuthCode(ctx context.Context, code string, meta RequestMeta) (*LoginResult, error) {
	var (
		userID    uuid.UUID
		ok        bool
		err       error
		u         model.User
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
	u, err = a.Users.GetID(ctx, userID)
	if err != nil {
		return nil, mapRepoNotFoundAs(err, ErrInvalidToken)
	}
	if !u.CanLogin() {
		return nil, ErrInvalidToken
	}
	authToken, err = a.Tokens.Issue(AuthClaims{
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

func (a *AuthUseCase) resolveGoogleUser(ctx context.Context, profile *GoogleProfile) (model.User, error) {
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
	)

	acc, err = a.OAuth.GetSub(ctx, model.OAuthProviderGoogle, profile.Sub)
	if isRepoNotFound(err) {
		hasAcc = false
	} else if err != nil {
		return model.User{}, err
	} else {
		hasAcc = true
	}
	if hasAcc {
		u, err = a.Users.GetID(ctx, acc.UserID)
		if err != nil {
			return model.User{}, mapRepoNotFound(err)
		}
		return u, nil
	}
	existing, err = a.Users.GetEmail(ctx, profile.Email)
	if isRepoNotFound(err) {
		existing = model.User{}
	} else if err != nil {
		return model.User{}, err
	}
	now = time.Now().UTC()
	if existing.ID != uuid.Nil {
		if existing.Role != model.UserRoleBuyer {
			return model.User{}, ErrOAuthForbidden
		}
		link = &model.OAuthAccount{
			ID:          uuid.New(),
			UserID:      existing.ID,
			Provider:    model.OAuthProviderGoogle,
			ProviderSub: profile.Sub,
			Email:       profile.Email,
			CreatedAt:   now,
		}
		if err = a.OAuth.Create(ctx, link); err != nil {
			return model.User{}, err
		}
		return existing, nil
	}
	pwPlain, err = OpaqueHex()
	if err != nil {
		return model.User{}, err
	}
	pwHash, err = HashPassword(pwPlain)
	if err != nil {
		return model.User{}, err
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
		AccountStatus: model.AccountStatusPending,
		SellerStatus:  model.SellerStatusNotApplicable,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err = a.Users.Create(ctx, &u); err != nil {
		return model.User{}, err
	}
	link = &model.OAuthAccount{
		ID:          uuid.New(),
		UserID:      u.ID,
		Provider:    model.OAuthProviderGoogle,
		ProviderSub: profile.Sub,
		Email:       profile.Email,
		CreatedAt:   now,
	}
	if err = a.OAuth.Create(ctx, link); err != nil {
		return model.User{}, fmt.Errorf("oauth link: %w", err)
	}
	return u, nil
}

const backupCodeCount = 10

type TOTPSetupResult struct {
	Secret     string
	OTPAuthURL string
}

type TOTPVerifyResult struct {
	BackupCodes []string
}

func (a *AuthUseCase) Setup2FA(ctx context.Context, userID uuid.UUID) (*TOTPSetupResult, error) {
	var (
		u      model.User
		err    error
		issuer string
		setup  *TOTPSetup
		enc    string
	)

	u, err = a.Users.GetID(ctx, userID)
	if err != nil {
		return nil, mapRepoNotFound(err)
	}
	if u.TOTPEnabled {
		return nil, Err2FAAlreadyEnabled
	}
	issuer = a.TOTPIssuer
	if issuer == "" {
		issuer = "Crowdfund"
	}
	setup, err = NewTOTPSetup(issuer, u.Email)
	if err != nil {
		return nil, err
	}
	enc, err = EncryptSecret(a.TOTPKey, setup.Secret)
	if err != nil {
		return nil, err
	}
	u.TOTPSecret = &enc
	u.UpdatedAt = time.Now().UTC()
	if err = a.Users.Update(ctx, &u); err != nil {
		return nil, err
	}
	return &TOTPSetupResult{Secret: setup.Secret, OTPAuthURL: setup.OTPAuthURL}, nil
}

func (a *AuthUseCase) Verify2FASetup(ctx context.Context, userID uuid.UUID, code string) (*TOTPVerifyResult, error) {
	var (
		u      model.User
		err    error
		secret string
		now    time.Time
		codes  []string
	)

	u, err = a.Users.GetID(ctx, userID)
	if err != nil {
		return nil, mapRepoNotFound(err)
	}
	if u.TOTPEnabled {
		return nil, Err2FAAlreadyEnabled
	}
	secret, err = a.decryptTOTP(u)
	if err != nil {
		return nil, Err2FANotConfigured
	}
	if !VerifyTOTP(secret, code) {
		return nil, ErrInvalid2FACode
	}
	now = time.Now().UTC()
	u.TOTPEnabled = true
	u.UpdatedAt = now
	if err = a.Users.Update(ctx, &u); err != nil {
		return nil, err
	}
	codes, err = a.genBackupCodes(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	if err = InvalidateUserSessions(ctx, a.DB, a.Users, a.RefreshTokens, u.ID); err != nil {
		return nil, err
	}
	return &TOTPVerifyResult{BackupCodes: codes}, nil
}

func (a *AuthUseCase) CreatePending2FAToken(_ context.Context, userID uuid.UUID) (string, error) {
	return SignPending2FA(a.pendingSecret, userID, time.Now().UTC()), nil
}

func (a *AuthUseCase) Challenge2FA(ctx context.Context, pendingToken, code string, meta RequestMeta) (*LoginResult, error) {
	var (
		uid    uuid.UUID
		err    error
		u      model.User
		tokens *SessionTokens
	)

	uid, err = VerifyPending2FA(a.pendingSecret, pendingToken, time.Now().UTC())
	if err != nil {
		return nil, ErrInvalidToken
	}
	u, err = a.Users.GetID(ctx, uid)
	if err != nil {
		return nil, mapRepoNotFoundAs(err, ErrInvalidToken)
	}
	if !u.CanLogin() {
		return nil, ErrInvalidToken
	}
	if !u.TOTPEnabled {
		return nil, Err2FANotConfigured
	}
	if !a.verifyTOTPOrBackup(ctx, u, code) {
		a.logActivity(ctx, &u.ID, model.UserLoginFail, "user", u.ID.String(), meta, model.StepMeta("2fa_fail"))
		return nil, ErrInvalid2FACode
	}
	tokens, err = a.issueSession(ctx, u, true)
	if err != nil {
		return nil, err
	}
	a.logActivity(ctx, &u.ID, model.UserLogin, "user", u.ID.String(), meta, model.StepMeta("2fa_ok"))
	return &LoginResult{
		User:         UserViewFrom(u),
		AuthToken:    tokens.AuthToken,
		RefreshToken: tokens.RefreshToken,
	}, nil
}

func (a *AuthUseCase) verifyTOTPOrBackup(ctx context.Context, u model.User, code string) bool {
	var (
		secret string
		err    error
		norm   string
		rows   []model.TOTPBackupCode
		hash   string
		now    time.Time
		i      int
		row    model.TOTPBackupCode
	)

	secret, err = a.decryptTOTP(u)
	if err == nil && VerifyTOTP(secret, code) {
		return true
	}
	if a.TOTPBackup == nil {
		return false
	}
	norm = strings.ToUpper(strings.TrimSpace(code))
	if norm == "" {
		return false
	}
	rows, err = a.TOTPBackup.ListUnused(ctx, u.ID)
	if err != nil {
		return false
	}
	hash = OpaqueHash(norm)
	now = time.Now().UTC()
	for i = range rows {
		row = rows[i]
		if row.CodeHash == hash {
			_ = a.TOTPBackup.MarkUsed(ctx, row.ID, now)
			return true
		}
	}
	return false
}

func (a *AuthUseCase) decryptTOTP(u model.User) (string, error) {
	if u.TOTPSecret == nil || strings.TrimSpace(*u.TOTPSecret) == "" {
		return "", Err2FANotConfigured
	}
	return DecryptSecret(a.TOTPKey, *u.TOTPSecret)
}

func (a *AuthUseCase) genBackupCodes(ctx context.Context, userID uuid.UUID) ([]string, error) {
	var (
		now   time.Time
		plain []string
		rows  []model.TOTPBackupCode
		i     int
		code  string
		err   error
	)

	if a.TOTPBackup == nil {
		return nil, nil
	}
	_ = a.TOTPBackup.DeleteUser(ctx, userID)
	now = time.Now().UTC()
	plain = make([]string, 0, backupCodeCount)
	rows = make([]model.TOTPBackupCode, 0, backupCodeCount)
	for i = 0; i < backupCodeCount; i++ {
		code, err = randBackupCode()
		if err != nil {
			return nil, err
		}
		plain = append(plain, code)
		rows = append(rows, model.TOTPBackupCode{
			ID:        uuid.New(),
			UserID:    userID,
			CodeHash:  OpaqueHash(code),
			CreatedAt: now,
		})
	}
	if err = a.TOTPBackup.CreateBatch(ctx, rows); err != nil {
		return nil, err
	}
	return plain, nil
}

func randBackupCode() (string, error) {
	var (
		b   []byte
		err error
	)

	b = make([]byte, 4)
	if _, err = rand.Read(b); err != nil {
		return "", err
	}
	return strings.ToUpper(hex.EncodeToString(b)), nil
}
