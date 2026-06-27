package usecase

import (
	"context"
	"time"

	"crowdfund-app/entity"
	"crowdfund-app/repository"
	"crowdfund-app/stripe"

	"github.com/google/uuid"
)

type ConnectUseCase struct {
	Users   repository.UserRepo
	Connect repository.ConnectRepo
	Stripe  stripe.Client
}

func (c *ConnectUseCase) GetAccount(ctx context.Context, sellerID uuid.UUID) (*ConnectAccountView, error) {
	var (
		u   *entity.User
		row *entity.ConnectAccount
		err error
	)
	if c.Stripe == nil {
		return nil, ErrStripeUnavailable
	}
	u, err = c.Users.FindID(ctx, sellerID)
	if err != nil || u == nil {
		return nil, ErrNotFound
	}
	if u.Role != entity.UserRoleSeller {
		return nil, ErrForbidden
	}
	row, err = c.Connect.FindUserID(ctx, sellerID)
	if err != nil {
		return nil, err
	}
	return ConnectAccountViewFrom(row), nil
}

func (c *ConnectUseCase) CreateOnboardingLink(ctx context.Context, sellerID uuid.UUID) (string, error) {
	var (
		u    *entity.User
		row  *entity.ConnectAccount
		url  string
		now  time.Time
		err  error
		snap stripe.AccountSnapshot
	)

	if c.Stripe == nil {
		return "", ErrStripeUnavailable
	}
	u, err = c.Users.FindID(ctx, sellerID)
	if err != nil || u == nil {
		return "", ErrNotFound
	}
	if u.Role != entity.UserRoleSeller || u.SellerStatus != entity.SellerStatusActive {
		return "", ErrForbidden
	}
	row, err = c.Connect.FindUserID(ctx, sellerID)
	if err != nil {
		return "", err
	}
	now = time.Now().UTC()
	if row == nil {
		var accountID string
		accountID, err = c.Stripe.CreateExpressAccount(ctx, u.Email)
		if err != nil {
			return "", err
		}
		row = &entity.ConnectAccount{
			ID:              uuid.New(),
			UserID:          sellerID,
			StripeAccountID: accountID,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		if err = c.Connect.Create(ctx, row); err != nil {
			return "", err
		}
	}
	snap, err = c.Stripe.GetAccount(ctx, row.StripeAccountID)
	if err != nil {
		return "", err
	}
	row.ChargesEnabled = snap.ChargesEnabled
	row.PayoutsEnabled = snap.PayoutsEnabled
	row.DetailsSubmitted = snap.DetailsSubmitted
	row.UpdatedAt = now
	if err = c.Connect.Update(ctx, row); err != nil {
		return "", err
	}
	url, err = c.Stripe.CreateAccountLink(ctx, row.StripeAccountID, c.Stripe.ConnectReturnURL(), c.Stripe.ConnectRefreshURL())
	return url, err
}

func (c *ConnectUseCase) SyncFromStripeAccountID(ctx context.Context, stripeAccountID string) (*ConnectAccountView, error) {
	var (
		row  *entity.ConnectAccount
		snap stripe.AccountSnapshot
		now  time.Time
		err  error
	)
	if c.Stripe == nil {
		return nil, ErrStripeUnavailable
	}
	row, err = c.Connect.FindStripeAccountID(ctx, stripeAccountID)
	if err != nil || row == nil {
		return nil, ErrNotFound
	}
	snap, err = c.Stripe.GetAccount(ctx, stripeAccountID)
	if err != nil {
		return nil, err
	}
	now = time.Now().UTC()
	row.ChargesEnabled = snap.ChargesEnabled
	row.PayoutsEnabled = snap.PayoutsEnabled
	row.DetailsSubmitted = snap.DetailsSubmitted
	row.UpdatedAt = now
	if err = c.Connect.Update(ctx, row); err != nil {
		return nil, err
	}
	return ConnectAccountViewFrom(row), nil
}
