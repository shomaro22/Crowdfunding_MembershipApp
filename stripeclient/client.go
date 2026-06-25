package stripeclient

import (
	"context"
	"strings"

	stripe "github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/account"
	"github.com/stripe/stripe-go/v82/accountlink"
	"github.com/stripe/stripe-go/v82/webhook"
)

type AccountSnapshot struct {
	ID               string
	ChargesEnabled   bool
	PayoutsEnabled   bool
	DetailsSubmitted bool
}

type Client interface {
	CreateExpressAccount(ctx context.Context, email string) (string, error)
	CreateAccountLink(ctx context.Context, accountID, returnURL, refreshURL string) (string, error)
	GetAccount(ctx context.Context, accountID string) (AccountSnapshot, error)
	ConstructWebhookEvent(payload []byte, signature string) (stripe.Event, error)
	ConnectReturnURL() string
	ConnectRefreshURL() string
}

type client struct {
	cfg Config
}

func New(cfg Config) Client {
	stripe.Key = cfg.SecretKey
	return &client{cfg: cfg}
}

func (c *client) CreateExpressAccount(ctx context.Context, email string) (string, error) {
	var (
		params *stripe.AccountParams
		acct   *stripe.Account
		err    error
	)
	_ = ctx
	params = &stripe.AccountParams{
		Type:    stripe.String(string(stripe.AccountTypeExpress)),
		Email:   stripe.String(email),
		Country: stripe.String("JP"),
		Capabilities: &stripe.AccountCapabilitiesParams{
			CardPayments: &stripe.AccountCapabilitiesCardPaymentsParams{
				Requested: stripe.Bool(true),
			},
			Transfers: &stripe.AccountCapabilitiesTransfersParams{
				Requested: stripe.Bool(true),
			},
		},
	}
	acct, err = account.New(params)
	if err != nil {
		return "", err
	}
	return acct.ID, nil
}

func (c *client) CreateAccountLink(ctx context.Context, accountID, returnURL, refreshURL string) (string, error) {
	var (
		params *stripe.AccountLinkParams
		link   *stripe.AccountLink
		err    error
	)
	_ = ctx
	params = &stripe.AccountLinkParams{
		Account:    stripe.String(accountID),
		ReturnURL:  stripe.String(returnURL),
		RefreshURL: stripe.String(refreshURL),
		Type:       stripe.String(string(stripe.AccountLinkTypeAccountOnboarding)),
	}
	link, err = accountlink.New(params)
	if err != nil {
		return "", err
	}
	return link.URL, nil
}

func (c *client) GetAccount(ctx context.Context, accountID string) (AccountSnapshot, error) {
	var (
		acct *stripe.Account
		err  error
		out  AccountSnapshot
	)
	_ = ctx
	acct, err = account.GetByID(accountID, nil)
	if err != nil {
		return AccountSnapshot{}, err
	}
	out = AccountSnapshot{
		ID:               acct.ID,
		ChargesEnabled:   acct.ChargesEnabled,
		PayoutsEnabled:   acct.PayoutsEnabled,
		DetailsSubmitted: acct.DetailsSubmitted,
	}
	return out, nil
}

func (c *client) ConstructWebhookEvent(payload []byte, signature string) (stripe.Event, error) {
	if strings.TrimSpace(c.cfg.WebhookSecret) == "" {
		return stripe.Event{}, ErrWebhookNotConfigured
	}
	return webhook.ConstructEvent(payload, signature, c.cfg.WebhookSecret)
}

func (c *client) ConnectReturnURL() string {
	return c.cfg.ConnectReturn
}

func (c *client) ConnectRefreshURL() string {
	return c.cfg.ConnectRefresh
}
