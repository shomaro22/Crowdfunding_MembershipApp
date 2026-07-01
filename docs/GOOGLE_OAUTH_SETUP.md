# Google OAuth 設定手順

FanFunding で「Google でログイン」を有効にするには、**Google Cloud Console 側の設定**と **本アプリ（backend）の環境変数**の両方が必要です。

## 1. Google Cloud Console

1. [Google Cloud Console](https://console.cloud.google.com/) でプロジェクトを作成（または既存を選択）
2. **API とサービス → OAuth 同意画面** を開き、外部/内部ユーザータイプを設定し、アプリ名・サポートメールを入力して保存
3. **API とサービス → 認証情報 → 認証情報を作成 → OAuth クライアント ID**
4. アプリケーションの種類: **ウェブアプリケーション**
5. **承認済みのリダイレクト URI** に以下を追加（利用環境に合わせて）:

| 環境 | リダイレクト URI |
|------|------------------|
| Docker（nginx 経由） | `http://localhost/auth/google/callback` |
| バックエンド直アクセス | `http://localhost:8080/auth/google/callback` |

6. 作成後、**クライアント ID** と **クライアント シークレット** を控える

## 2. 本リポジトリの環境変数（backend）

`.env` または `docker-compose.yml` の `backend` サービスに設定:

```env
GOOGLE_CLIENT_ID=xxxxxxxx.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=GOCSPX-xxxxxxxx
GOOGLE_REDIRECT_URI=http://localhost/auth/google/callback
PUBLIC_APP_URL=http://localhost
API_PUBLIC_URL=http://localhost
CORS_ALLOW_ORIGINS=http://localhost,http://127.0.0.1
REDIS_ADDR=redis:6379
```

## 3. 動作確認

1. `docker compose up -d --build`
2. `http://localhost/signin` → **Google でログイン**
3. Google アカウント選択 → 同意 → ログイン完了

## 4. トラブルシューティング

| 症状 | 対処 |
|------|------|
| redirect_uri_mismatch | Console の URI と `GOOGLE_REDIRECT_URI` を完全一致させる |
| OAuth ルートが無い | `GOOGLE_CLIENT_ID` / `SECRET` 未設定時は backend が OAuth を無効化 |
| WebSocket 失敗 | `CORS_ALLOW_ORIGINS` に `http://localhost` を含める |
