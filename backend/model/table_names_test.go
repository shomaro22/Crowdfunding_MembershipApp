package model

import "testing"

func TestAllActivityActionsValid(t *testing.T) {
	var actions = []ActivityAction{
		UserLogin, UserLoginFail, UserLogout, UserRegister, SellerRegister,
		FavoriteAdd, FavoriteRemove,
		PurchaseCreated, PurchaseCancelled, PurchasePlanChange, PurchaseRefunded, PurchaseDisputed,
		MessageSent, MessageRead,
		ProjectCreated, ProjectUpdated,
		PaymentCheckout, PaymentWebhook,
		ReportCreated,
		SecRateLimit, SecWSConnect, SecWSSubscribe, Sec2FAOn, Sec2FAFail,
	}

	var i int
	var a ActivityAction
	for i = range actions {
		a = actions[i]
		if !a.Valid() {
			t.Fatalf("action %q should be valid", a)
		}
	}
	if ActivityAction("bad.action").Valid() {
		t.Fatal("invalid action should fail")
	}
}

func TestModelTableNames(t *testing.T) {
	type namer interface {
		TableName() string
	}

	cases := []struct {
		model namer
		want  string
	}{
		{User{}, "users"},
		{ConnectAccount{}, "connect_accounts"},
		{ActivityLog{}, "activity_logs"},
		{AuditLog{}, "audit_logs"},
		{BatchRun{}, "batch_job_runs"},
		{Favorite{}, "favorites"},
		{Message{}, "messages"},
		{OAuthAccount{}, "oauth_accounts"},
		{Project{}, "projects"},
		{ProjectPlan{}, "project_plans"},
		{ProjectImage{}, "project_images"},
		{Purchase{}, "purchases"},
		{PurchaseEvent{}, "purchase_events"},
		{ProjectSupporter{}, "project_supporters"},
		{SellerReg{}, "seller_registrations"},
		{Report{}, "reports"},
		{RefreshToken{}, "refresh_tokens"},
		{ResetToken{}, "password_reset_tokens"},
		{WSToken{}, "ws_tokens"},
		{TOTPBackupCode{}, "totp_backup_codes"},
		{WebhookEvent{}, "stripe_webhook_events"},
		{ReviewDecision{}, "review_decisions"},
	}

	var i int
	var c struct {
		model namer
		want  string
	}
	for i = range cases {
		c = cases[i]
		if c.model.TableName() != c.want {
			t.Fatalf("TableName()=%q want %q", c.model.TableName(), c.want)
		}
	}
}
