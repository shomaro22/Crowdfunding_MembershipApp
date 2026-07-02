# Runbook: OAuth 障害

## 症状

- `/auth/google` が 503
- コールバックで `oauth_failed`

## 初動

1. `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET` / `GOOGLE_REDIRECT_URI` を確認
2. Redis が起動しているか確認（OAuth state 保存）
3. `PUBLIC_APP_URL` とフロント nginx の `/auth/callback` 設定を確認

## Google Cloud Console

- 承認済みリダイレクト URI に `{API_PUBLIC_URL}/auth/google/callback` が登録されていること
- PKCE 有効化後も Authorization Code フローが許可されていること
