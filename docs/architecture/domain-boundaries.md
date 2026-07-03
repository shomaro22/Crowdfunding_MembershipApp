# ドメイン境界

## Project（プロジェクト）

- 出品者が作成・更新するクラウドファンディング案件
- 審査・公開状態は Admin ドメインと連携
- 支援・決済の対象エンティティ

## Support（支援）

- 購入者のプラン選択・支援申込
- Supporter / Purchase レコードのライフサイクル
- プラン変更・解約・再参加

## Payment（決済）

- Stripe Checkout / PaymentIntent
- Webhook 処理・購入確定
- Connect（出品者入金）

## Membership（メンバーシップ）

- 支援確定後の Supporter 関係
- プラン特典・メッセージアクセス権

## Notification（通知）

- WebSocket メッセージ配信
- activity_log による監査・セキュリティイベント

## イベント連携（将来）

`SupportCompleted` イベントで Notification / Mail / Activity を非同期連携する設計を推奨。
