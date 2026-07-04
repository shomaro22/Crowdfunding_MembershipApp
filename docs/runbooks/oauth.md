# Runbook: OAuth / OIDC 障害

フロー詳細: [../architecture/google-oauth-oidc.md](../architecture/google-oauth-oidc.md)

## 症状

| 症状 | 想定原因 |
|------|----------|
| `GET /auth/google` が 503/404 | `GOOGLE_CLIENT_ID` / `SECRET` 未設定 |
| コールバック `oauth_failed` | state 失効、PKCE 不一致、Redis 障害 |
| `redirect_uri_mismatch` | Console URI ≠ `GOOGLE_REDIRECT_URI` |
| ログインループ | `/auth/refresh` が 204 → Cookie 無効（正常動作） |
| 連携失敗 | 既存アカウントでパスワード誤り |

## 初動チェックリスト

1. **環境変数**
   - `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GOOGLE_REDIRECT_URI`
   - `PUBLIC_APP_URL`（フロント）, `API_PUBLIC_URL`（API）
   - `CORS_ALLOW_ORIGINS`
2. **Redis**
   - `/health` で `redis: ok`
   - OAuth state は Redis に 10 分 TTL で保存
   - 本番: `REDIS_URL`（Render Key Value）または `REDIS_ADDR` + `REDIS_PASSWORD`
3. **リダイレクト**
   - Google → `{API}/auth/google/callback`
   - API → `{PUBLIC_APP_URL}/auth/callback?code=...`
4. **Cookie**
   - 本番 `COOKIE_SECURE=true`（HTTPS 必須）
   - フロントと API が別ドメインの場合 CORS + `credentials: include`

## Google Cloud Console

- 承認済みリダイレクト URI: `{API_PUBLIC_URL}/auth/google/callback`
- Authorization Code フロー（PKCE S256）が許可されていること
- OAuth 同意画面が「テスト」モードの場合、テストユーザーに Google アカウントを追加

## 本番（Render + Vercel）例

```text
GOOGLE_REDIRECT_URI=https://crowdfunding-membershipapp.onrender.com/auth/google/callback
PUBLIC_APP_URL=https://crowdfunding-membership-app.vercel.app
API_PUBLIC_URL=https://crowdfunding-membershipapp.onrender.com
CORS_ALLOW_ORIGINS=https://crowdfunding-membership-app.vercel.app
```

## ログ確認

```bash
# ローカル
docker compose logs backend | tail -100

# Render
Dashboard → Logs（OAuth エラー・Redis 接続エラー）
```

## 復旧後の確認

1. `curl -s https://<api>/health` → db/redis ok
2. ブラウザで Google ログイン → `/auth/callback` → ホーム表示
3. 既存メールで Google ログイン → 連携確認画面 → 成功
