package stripe

import (
	"context"
	"strings"

	stripesdk "github.com/stripe/stripe-go/v82"
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
	ConstructWebhookEvent(payload []byte, signature string) (WebhookEvent, error)
	ConnectReturnURL() string
	ConnectRefreshURL() string
	AppURL() string
	CreateCheckoutSession(ctx context.Context, in CheckoutSessionIn) (CheckoutSessionOut, error)
	CreateSubscriptionIntent(ctx context.Context, in SubscriptionIntentIn) (SubscriptionIntentOut, error)
	CancelSubscriptionAtPeriodEnd(ctx context.Context, subID string) error
	ResumeSubscription(ctx context.Context, subID string) error
	UpdateSubscriptionAmount(ctx context.Context, subID string, amount int64) error
	CreateUpgradePaymentIntent(ctx context.Context, in UpgradePaymentIn) (UpgradePaymentOut, error)
	GetPaymentIntent(ctx context.Context, intentID string) (PaymentIntentSnapshot, error)
	GetCheckoutSession(ctx context.Context, sessionID string) (CheckoutSessionSnapshot, error)
	GetPaymentIntentDetail(ctx context.Context, intentID string) (PaymentIntentDetail, error)
}

type client struct {
	cfg Config
}

func New(cfg Config) Client {
	stripesdk.Key = cfg.SecretKey
	return &client{cfg: cfg}
}

func (c *client) CreateExpressAccount(ctx context.Context, email string) (string, error) {
	var (
		params *stripesdk.AccountParams
		acct   *stripesdk.Account
		err    error
	)
	_ = ctx
	params = &stripesdk.AccountParams{
		Type:    stripesdk.String(string(stripesdk.AccountTypeExpress)),
		Email:   stripesdk.String(email),
		Country: stripesdk.String("JP"),
		Capabilities: &stripesdk.AccountCapabilitiesParams{
			CardPayments: &stripesdk.AccountCapabilitiesCardPaymentsParams{
				Requested: stripesdk.Bool(true),
			},
			Transfers: &stripesdk.AccountCapabilitiesTransfersParams{
				Requested: stripesdk.Bool(true),
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
		params *stripesdk.AccountLinkParams
		link   *stripesdk.AccountLink
		err    error
	)
	_ = ctx
	params = &stripesdk.AccountLinkParams{
		Account:    stripesdk.String(accountID),
		ReturnURL:  stripesdk.String(returnURL),
		RefreshURL: stripesdk.String(refreshURL),
		Type:       stripesdk.String(string(stripesdk.AccountLinkTypeAccountOnboarding)),
	}
	link, err = accountlink.New(params)
	if err != nil {
		return "", err
	}
	return link.URL, nil
}

func (c *client) GetAccount(ctx context.Context, accountID string) (AccountSnapshot, error) {
	var (
		acct *stripesdk.Account
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

func (c *client) ConstructWebhookEvent(payload []byte, signature string) (WebhookEvent, error) {
	if strings.TrimSpace(c.cfg.WebhookSecret) == "" {
		return WebhookEvent{}, ErrWebhookNotConfigured
	}
	ev, err := webhook.ConstructEvent(payload, signature, c.cfg.WebhookSecret)
	if err != nil {
		return WebhookEvent{}, err
	}
	return WebhookEvent{
		ID:      ev.ID,
		Type:    string(ev.Type),
		DataRaw: ev.Data.Raw,
	}, nil
}

func (c *client) ConnectReturnURL() string {
	return c.cfg.ConnectReturn
}

func (c *client) ConnectRefreshURL() string {
	return c.cfg.ConnectRefresh
}

func (c *client) AppURL() string {
	return c.cfg.AppURL
}
