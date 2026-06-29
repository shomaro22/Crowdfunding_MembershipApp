package usecase

import (
	"context"
	"strings"
	"time"

	"crowdfund-app/auth"
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
		return UserView{}, mapBuyerRegEntityErr(err)
	}
	exists, err = a.Users.EmailExists(ctx, normalized)
	if err != nil {
		return UserView{}, err
	}
	if exists {
		return UserView{}, ErrEmailInUse
	}
	hash, err = auth.HashPassword(password)
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
	return UserViewFrom(u), nil
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
		return nil, mapBuyerRegEntityErr(err)
	}
	if err = validator.ValidateSellerKYC(in.FullName, in.BirthDate, in.Address, in.PhotoPresent); err != nil {
		return nil, mapSellerKYCEntityErr(err)
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

	hash, err = auth.HashPassword(in.Password)
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
	return &SellerRegOut{User: UserViewFrom(u), Registration: SellerRegItemFrom(reg)}, nil
}
