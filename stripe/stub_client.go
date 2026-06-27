package stripe

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"

	"github.com/google/uuid"
	stripesdk "github.com/stripe/stripe-go/v82"
)

// StubClient は統合テスト向けのインメモリ Stripe 実装（本番では使わない）。
const StubWebhookSignature = "whsec_test_sig"

type StubClient struct {
	mu       sync.Mutex
	cfg      Config
	sessions map[string]*stubSession
	intents  map[string]*stubIntent
}

type stubSession struct {
	out           CheckoutSessionOut
	in            CheckoutSessionIn
	paymentStatus string
}

type stubIntent struct {
	metadata map[string]string
	status   string
	subID    string
	custID   string
}

func NewStubClient() *StubClient {
	return &StubClient{
		cfg: Config{
			AppURL:        "http://localhost:5173",
			WebhookSecret: "whsec_test",
		},
		sessions: make(map[string]*stubSession),
		intents:  make(map[string]*stubIntent),
	}
}

// SetSessionPaid はテストから Checkout セッションを支払い済みにする。
func (c *StubClient) SetSessionPaid(sessionID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	s, ok := c.sessions[sessionID]
	if !ok {
		return
	}
	s.paymentStatus = CheckoutPaymentStatusPaid
	if s.out.PaymentIntentID != "" {
		if pi, ok := c.intents[s.out.PaymentIntentID]; ok {
			pi.status = PaymentIntentStatusSucceeded
		}
	}
}

// SetIntentSucceeded はテストから PaymentIntent を成功状態にする。
func (c *StubClient) SetIntentSucceeded(intentID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if pi, ok := c.intents[intentID]; ok {
		pi.status = PaymentIntentStatusSucceeded
	}
}

func (c *StubClient) SessionMetadata(sessionID string) map[string]string {
	c.mu.Lock()
	defer c.mu.Unlock()
	s, ok := c.sessions[sessionID]
	if !ok {
		return nil
	}
	return cloneMeta(s.in.Metadata)
}

func (c *StubClient) IntentMetadata(intentID string) map[string]string {
	c.mu.Lock()
	defer c.mu.Unlock()
	pi, ok := c.intents[intentID]
	if !ok {
		return nil
	}
	return cloneMeta(pi.metadata)
}

func cloneMeta(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func (c *StubClient) nextCheckoutIDs() (string, string) {
	suffix := strings.ReplaceAll(uuid.New().String(), "-", "")
	return "cs_test_" + suffix, "pi_test_" + suffix
}

func (c *StubClient) CreateExpressAccount(ctx context.Context, email string) (string, error) {
	_ = ctx
	_ = email
	return "acct_test_stub", nil
}

func (c *StubClient) CreateAccountLink(ctx context.Context, accountID, returnURL, refreshURL string) (string, error) {
	_ = ctx
	_ = accountID
	_ = returnURL
	_ = refreshURL
	return "https://connect.test/onboard", nil
}

func (c *StubClient) GetAccount(ctx context.Context, accountID string) (AccountSnapshot, error) {
	_ = ctx
	return AccountSnapshot{
		ID:               accountID,
		ChargesEnabled:   true,
		PayoutsEnabled:   true,
		DetailsSubmitted: true,
	}, nil
}

func (c *StubClient) ConstructWebhookEvent(payload []byte, signature string) (WebhookEvent, error) {
	if signature != StubWebhookSignature {
		return WebhookEvent{}, errors.New("invalid stripe signature")
	}
	var ev stripesdk.Event
	if err := json.Unmarshal(payload, &ev); err != nil {
		return WebhookEvent{}, err
	}
	return WebhookEvent{
		ID:      ev.ID,
		Type:    string(ev.Type),
		DataRaw: ev.Data.Raw,
	}, nil
}

func (c *StubClient) ConnectReturnURL() string  { return c.cfg.ConnectReturn }
func (c *StubClient) ConnectRefreshURL() string { return c.cfg.ConnectRefresh }
func (c *StubClient) AppURL() string            { return c.cfg.AppURL }

func (c *StubClient) CreateCheckoutSession(ctx context.Context, in CheckoutSessionIn) (CheckoutSessionOut, error) {
	_ = ctx
	c.mu.Lock()
	defer c.mu.Unlock()
	var (
		sid, pid = c.nextCheckoutIDs()
		out      CheckoutSessionOut
	)
	out = CheckoutSessionOut{
		SessionID:       sid,
		URL:             "https://checkout.test/" + sid,
		PaymentIntentID: pid,
	}
	c.sessions[sid] = &stubSession{
		out:           out,
		in:            in,
		paymentStatus: CheckoutPaymentStatusUnpaid,
	}
	c.intents[pid] = &stubIntent{
		metadata: cloneMeta(in.Metadata),
		status:   "requires_payment_method",
	}
	return out, nil
}

func (c *StubClient) CreateSubscriptionIntent(ctx context.Context, in SubscriptionIntentIn) (SubscriptionIntentOut, error) {
	_ = ctx
	c.mu.Lock()
	defer c.mu.Unlock()
	var (
		suffix = strings.ReplaceAll(uuid.New().String(), "-", "")
		piID   = "pi_test_" + suffix
		subID  = "sub_test_" + suffix
		custID = "cus_test_" + suffix
	)
	c.intents[piID] = &stubIntent{
		metadata: cloneMeta(in.Metadata),
		status:   "requires_payment_method",
		subID:    subID,
		custID:   custID,
	}
	return SubscriptionIntentOut{
		SubscriptionID:  subID,
		PaymentIntentID: piID,
		ClientSecret:    "cs_test_secret_" + piID,
		CustomerID:      custID,
	}, nil
}

func (c *StubClient) CancelSubscriptionAtPeriodEnd(ctx context.Context, subID string) error {
	_ = ctx
	_ = subID
	return nil
}

func (c *StubClient) ResumeSubscription(ctx context.Context, subID string) error {
	_ = ctx
	_ = subID
	return nil
}

func (c *StubClient) UpdateSubscriptionAmount(ctx context.Context, subID string, amount int64) error {
	_ = ctx
	_ = subID
	_ = amount
	return nil
}

func (c *StubClient) CreateUpgradePaymentIntent(ctx context.Context, in UpgradePaymentIn) (UpgradePaymentOut, error) {
	_ = ctx
	c.mu.Lock()
	defer c.mu.Unlock()
	var piID = "pi_test_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	c.intents[piID] = &stubIntent{
		metadata: cloneMeta(in.Metadata),
		status:   "requires_payment_method",
		custID:   in.CustomerID,
	}
	return UpgradePaymentOut{
		PaymentIntentID: piID,
		ClientSecret:    "cs_test_secret_" + piID,
	}, nil
}

func (c *StubClient) GetPaymentIntent(ctx context.Context, intentID string) (PaymentIntentSnapshot, error) {
	_ = ctx
	c.mu.Lock()
	defer c.mu.Unlock()
	pi, ok := c.intents[intentID]
	if !ok {
		return PaymentIntentSnapshot{}, errors.New("payment intent not found")
	}
	return PaymentIntentSnapshot{ID: intentID, Status: pi.status}, nil
}

func (c *StubClient) GetCheckoutSession(ctx context.Context, sessionID string) (CheckoutSessionSnapshot, error) {
	_ = ctx
	c.mu.Lock()
	defer c.mu.Unlock()
	s, ok := c.sessions[sessionID]
	if !ok {
		return CheckoutSessionSnapshot{}, errors.New("checkout session not found")
	}
	pm := []string{"card"}
	if s.in.PaymentMethod == "konbini" {
		pm = []string{"konbini"}
	}
	return CheckoutSessionSnapshot{
		ID:                 sessionID,
		PaymentStatus:      s.paymentStatus,
		Metadata:           cloneMeta(s.in.Metadata),
		PaymentMethodTypes: pm,
	}, nil
}

func (c *StubClient) GetPaymentIntentDetail(ctx context.Context, intentID string) (PaymentIntentDetail, error) {
	_ = ctx
	c.mu.Lock()
	defer c.mu.Unlock()
	pi, ok := c.intents[intentID]
	if !ok {
		return PaymentIntentDetail{}, errors.New("payment intent not found")
	}
	return PaymentIntentDetail{
		ID:       intentID,
		Status:   pi.status,
		Metadata: cloneMeta(pi.metadata),
	}, nil
}
