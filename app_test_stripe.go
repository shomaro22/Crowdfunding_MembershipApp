package main

import "crowdfund-app/stripe"

// 統合テスト専用: buildApp に Stub Stripe を差し替える。
var testStripeClient stripe.Client

func setTestStripeClient(c stripe.Client) { testStripeClient = c }

func clearTestStripeClient() { testStripeClient = nil }
