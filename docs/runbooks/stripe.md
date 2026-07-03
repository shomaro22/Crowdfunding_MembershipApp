# Runbook: Stripe 障害

## 症状

- 決済 API が 503 / `payment_unavailable`
- Webhook が未処理のまま滞留

## 初動

1. [Stripe Status](https://status.stripe.com/) を確認
2. `STRIPE_SECRET_KEY` / `STRIPE_WEBHOOK_SECRET` の有効性を確認
3. 管理画面の購入一覧で `pending` の滞留を確認

## ローカル開発

`STRIPE_USE_STUB=1` でスタブ決済に切り替え可能。

## Webhook 再送

Stripe Dashboard → Developers → Webhooks → 失敗イベントを Resend。
