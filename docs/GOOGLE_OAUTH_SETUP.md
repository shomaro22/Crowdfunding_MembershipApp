# Google OAuth 設定手順

FanFunding の Google ログインは **バックエンド主導の OIDC（Authorization Code + PKCE）** です。フロントの `react-oidc-context` は **使いません**。

詳細フロー: [docs/architecture/google-oauth-oidc.md](architecture/google-oauth-oidc.md)

## 1. Google Cloud Console

1. [Google Cloud Console](https://console.cloud.google.com/) でプロジェクトを作成（または既存を選択）
2. **API とサービス → OAuth 同意画面** — アプリ名・サポートメールを設定
3. **API とサービス → 認証情報 → OAuth クライアント ID**（種類: **ウェブアプリケーション**）
4. **承認済みのリダイレクト URI**（`GOOGLE_REDIRECT_URI` と **完全一致**）:

| 環境 | リダイレクト URI |
|------|------------------|
| Docker（nginx 経由） | `http://localhost/auth/google/callback` |
| バックエンド直（開発） | `http://localhost:8080/auth/google/callback` |
| 本番（Render API） | `https://<your-api>.onrender.com/auth/google/callback` |

5. クライアント ID / シークレットを控える

## 2. 環境変数（backend）

### ローカル（docker-compose）

```env
GOOGLE_CLIENT_ID=xxxxxxxx.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=GOCSPX-xxxxxxxx
GOOGLE_REDIRECT_URI=http://localhost/auth/google/callback
PUBLIC_APP_URL=http://localhost
API_PUBLIC_URL=http://localhost
CORS_ALLOW_ORIGINS=http://localhost,http://127.0.0.1,http://localhost:5173
REDIS_ADDR=redis:6379
```

### 本番（Render + Vercel）

```env
GOOGLE_CLIENT_ID=...
GOOGLE_CLIENT_SECRET=...
GOOGLE_REDIRECT_URI=https://<your-api>.onrender.com/auth/google/callback
PUBLIC_APP_URL=https://<your-app>.vercel.app
API_PUBLIC_URL=https://<your-api>.onrender.com
CORS_ALLOW_ORIGINS=https://<your-app>.vercel.app
REDIS_URL=redis://...   # Render Key Value 連携可
COOKIE_SECURE=true
```

フロント（Vercel）:

```env
VITE_API_BASE_URL=https://<your-api>.onrender.com
```

## 3. 動作確認

1. `docker compose up -d --build`（ローカル）または本番デプロイ後
2. `/signin` → **Google でログイン**
3. Google 同意 → `/auth/callback` → ログイン完了
4. 既存メールがある場合 → `/oauth/link-confirm` でパスワード確認

## 4. トラブルシューティング

| 症状 | 対処 |
|------|------|
| `redirect_uri_mismatch` | Console の URI と `GOOGLE_REDIRECT_URI` を完全一致 |
| `/auth/google` が 404/503 | `GOOGLE_CLIENT_ID` / `SECRET` 未設定 → OAuth ルート未登録 |
| `oauth_failed` / state エラー | **Redis** 起動・`REDIS_URL` 確認 |
| ログイン後すぐログアウト | `CORS_ALLOW_ORIGINS` / `COOKIE_SECURE` / 同一サイト Cookie 設定 |
| 連携画面が出ない | 既存 Credentials と Google email が一致 → `/oauth/link-confirm` |
| WebSocket 失敗 | CORS にフロントオリジンを含める |

障害 Runbook: [docs/runbooks/oauth.md](runbooks/oauth.md)
