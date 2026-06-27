package usecase

import (
	"context"
	"strings"
	"time"

	"crowdfund-app/auth"
	"crowdfund-app/entity"
	"crowdfund-app/repository"

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

func (a *AuthUseCase) GetUser(ctx context.Context, userID uuid.UUID) (UserView, error) {
	var (
		u   *entity.User
		err error
	)

	u, err = a.Users.FindID(ctx, userID)
	if err != nil {
		return UserView{}, err
	}
	if u == nil {
		return UserView{}, ErrNotFound
	}
	return UserViewFrom(u), nil
}

func (a *AuthUseCase) ParseToken(raw string) (auth.AuthClaims, error) {
	return a.Tokens.Parse(raw)
}

func (a *AuthUseCase) SigningKey() []byte {
	return a.TOTPKey
}

func (a *AuthUseCase) RegisterSeller(ctx context.Context, in SellerRegIn, meta AuthMeta) (*SellerRegOut, error) {
	var (
		normalized string
		err        error
		exists     bool
		userID     uuid.UUID
		frontKey   string
		backKey    string
		hash       string
		now        time.Time
		u          *entity.User
		reg        *entity.SellerReg
	)

	if a.KYCStore == nil || a.SellerRegs == nil || a.DB == nil {
		return nil, ErrNotFound
	}

	normalized, err = entity.ValidateBuyerReg(in.Email, in.Password)
	if err != nil {
		switch err {
		case entity.ErrBadEmail:
			return nil, ErrInvalidEmail
		case entity.ErrWeakPassword:
			return nil, ErrWeakPassword
		default:
			return nil, err
		}
	}
	if err = entity.ValidateSellerKYC(in.FullName, in.BirthDate, in.Address, in.PhotoPresent); err != nil {
		return nil, err
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
		return nil, mapKYCStoreError(err)
	}
	backKey, err = a.KYCStore.Save(ctx, userID, "back", in.Back.toStoreImage())
	if err != nil {
		_ = a.KYCStore.Remove(ctx, frontKey)
		return nil, mapKYCStoreError(err)
	}

	hash, err = auth.HashPassword(in.Password)
	if err != nil {
		_ = a.KYCStore.Remove(ctx, frontKey, backKey)
		return nil, err
	}

	now = time.Now().UTC()
	u = &entity.User{
		ID:            userID,
		Email:         normalized,
		PasswordHash:  hash,
		Role:          entity.UserRoleSeller,
		AccountStatus: entity.AccountStatusPending,
		SellerStatus:  entity.SellerStatusPending,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	reg = &entity.SellerReg{
		ID:           uuid.New(),
		UserID:       userID,
		FullName:     strings.TrimSpace(in.FullName),
		BirthDate:    in.BirthDate,
		Address:      strings.TrimSpace(in.Address),
		PhotoPresent: in.PhotoPresent,
		IDFrontKey:   &frontKey,
		IDBackKey:    &backKey,
		Status:       entity.RegPending,
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

	a.logActivity(ctx, &u.ID, entity.SellerRegister, "seller_registration", reg.ID.String(), meta, nil)
	return &SellerRegOut{User: UserViewFrom(u), Registration: SellerRegItemFrom(reg)}, nil
}
