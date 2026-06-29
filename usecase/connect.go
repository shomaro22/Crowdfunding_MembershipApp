package usecase

import (
	"context"
	"time"

	"crowdfund-app/model"
	"crowdfund-app/repository"
	"crowdfund-app/stripe"

	"github.com/google/uuid"
)

type ConnectUseCase struct {
	Users   repository.IUserRepo
	Connect repository.IConnectRepo
	Stripe  stripe.Client
}

func (c *ConnectUseCase) GetAccount(ctx context.Context, sellerID uuid.UUID) (*ConnectAccountView, error) {
	var (
		u   model.User
		row model.ConnectAccount
		err error
	)
	if c.Stripe == nil {
		return nil, ErrStripeUnavailable
	}
	u, err = c.Users.GetID(ctx, sellerID)
	if err != nil {
		return nil, mapRepoNotFound(err)
	}
	if u.Role != model.UserRoleSeller {
		return nil, ErrForbidden
	}
	row, err = c.Connect.GetUserID(ctx, sellerID)
	if isRepoNotFound(err) {
		return ConnectAccountViewFrom(model.ConnectAccount{}), nil
	}
	if err != nil {
		return nil, err
	}
	return ConnectAccountViewFrom(row), nil
}

func (c *ConnectUseCase) CreateOnboardingLink(ctx context.Context, sellerID uuid.UUID) (string, error) {
	var (
		u      model.User
		row    model.ConnectAccount
		hasRow bool
		url    string
		now    time.Time
		err    error
		snap   stripe.AccountSnapshot
	)

	if c.Stripe == nil {
		return "", ErrStripeUnavailable
	}
	u, err = c.Users.GetID(ctx, sellerID)
	if err != nil {
		return "", mapRepoNotFound(err)
	}
	if u.Role != model.UserRoleSeller || u.SellerStatus != model.SellerStatusActive {
		return "", ErrForbidden
	}
	row, err = c.Connect.GetUserID(ctx, sellerID)
	if isRepoNotFound(err) {
		hasRow = false
	} else if err != nil {
		return "", err
	} else {
		hasRow = true
	}
	now = time.Now().UTC()
	if !hasRow {
		var accountID string
		accountID, err = c.Stripe.CreateExpressAccount(ctx, u.Email)
		if err != nil {
			return "", err
		}
		row = model.ConnectAccount{
			ID:              uuid.New(),
			UserID:          sellerID,
			StripeAccountID: accountID,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		if err = c.Connect.Create(ctx, &row); err != nil {
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
	if err = c.Connect.Update(ctx, &row); err != nil {
		return "", err
	}
	url, err = c.Stripe.CreateAccountLink(ctx, row.StripeAccountID, c.Stripe.ConnectReturnURL(), c.Stripe.ConnectRefreshURL())
	return url, err
}

func (c *ConnectUseCase) SyncFromStripeAccountID(ctx context.Context, stripeAccountID string) (*ConnectAccountView, error) {
	var (
		row  model.ConnectAccount
		snap stripe.AccountSnapshot
		now  time.Time
		err  error
	)
	if c.Stripe == nil {
		return nil, ErrStripeUnavailable
	}
	row, err = c.Connect.GetStripeAccountID(ctx, stripeAccountID)
	if err != nil {
		return nil, mapRepoNotFound(err)
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
	if err = c.Connect.Update(ctx, &row); err != nil {
		return nil, err
	}
	return ConnectAccountViewFrom(row), nil
}

type ConnectSyncer interface {
	SyncFromStripeAccountID(ctx context.Context, stripeAccountID string) (*ConnectAccountView, error)
}

type SupporterViewer interface {
	ViewForSupporter(ctx context.Context, supporter model.ProjectSupporter) (SupporterView, error)
}
