package bootstrap

import "crowdfund-app/stripe"

// Options は BuildHTTP の任意設定（主に統合テスト用 Stripe 差し替え）。
type Options struct {
	Stripe stripe.Client
}
