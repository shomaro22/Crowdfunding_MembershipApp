# 本番デプロイ手順: Vercel（フロント） + Neon（DB）

このアプリは **フロント（React）** と **API（Go）** と **PostgreSQL** と **Redis** で動きます。

| コンポーネント | 推奨ホスト | 備考 |
|----------------|------------|------|
| フロントエンド | **Vercel** | `frontend/` を静的配信 |
| PostgreSQL | **Neon** | `DATABASE_URL` |
| API（Go） | Railway / Fly.io / Render 等 | Neon だけでは API は動かない |
| Redis | Upstash Redis 等 | OAuth 状態・レート制限に必須 |
| 画像 / KYC ファイル | API サーバーのディスク or 将来 S3 | コンテナ再起動で消える場合あり |

以下は最小構成の手順です。

---

## 0. 事前に決める URL

例（カスタムドメイン推奨）:

| 用途 | 例 |
|------|-----|
| フロント | `https://app.example.com` |
| API | `https://api.example.com` |

Vercel の `*.vercel.app` と API の別ドメインでも動かせますが、**Refresh Cookie** は同一サイト（同じ eTLD+1、例: `app.example.com` + `api.example.com`）の方が設定が楽です。

---

## 1. Neon（PostgreSQL）

1. [Neon](https://neon.tech/) でプロジェクト作成
2. **Connection string** をコピー（プール付き推奨）
3. 末尾に SSL を付ける（Neon 必須）:

```text
postgres://USER:PASSWORD@HOST/DBNAME?sslmode=require
```

4. ローカルからマイグレーション確認（任意）:

```bash
export DATABASE_URL='postgres://...?sslmode=require'
cd backend
go run .   # 起動時に db.Up() で migration 自動適用
# またはシードのみ:
go run ./cmd/seed dev -team
```

本番では **デモシード（`-team -reset`）は実行しない** でください。

---

## 2. Redis（Upstash 例）

1. [Upstash](https://upstash.com/) で Redis 作成
2. 接続情報を控える
3. バックエンド環境変数:

```env
REDIS_ADDR=xxxx.upstash.io:6379
REDIS_PASSWORD=xxxxxxxx
```

---

## 3. バックエンド API のデプロイ

Neon は DB のみです。**Go API は別サービス**に Docker イメージをデプロイします。

### 3.1 ビルド対象

リポジトリルートの `Dockerfile`、ターゲット `backend-runtime` を使用します。

```bash
docker build --target backend-runtime -t crowdfund-api .
```

### 3.2 必須環境変数（本番）

`.env.example` をベースに、最低限以下を設定します。

```env
# DB（Neon）
DATABASE_URL=postgres://...?sslmode=require

# Redis
REDIS_ADDR=...
REDIS_PASSWORD=...

# 公開 URL（実際のドメインに合わせる）
PORT=8080
API_PUBLIC_URL=https://api.example.com
PUBLIC_APP_URL=https://app.example.com
CORS_ALLOW_ORIGINS=https://app.example.com

# Cookie（HTTPS 必須）
COOKIE_SECURE=true

# 認証（各 32 文字以上・本番用にランダム生成・互いに別値）
JWT_SECRET=...
REFRESH_SECRET=...
TOTP_ENCRYPTION_KEY=...
KYC_SIGNING_KEY=...
PENDING_2FA_KEY=...

# Stripe（本番はスタブ OFF）
STRIPE_USE_STUB=0
STRIPE_SECRET_KEY=sk_live_...
STRIPE_WEBHOOK_SECRET=whsec_...
STRIPE_CONNECT_RETURN_URL=https://app.example.com/seller/connect/return
STRIPE_CONNECT_REFRESH_URL=https://app.example.com/seller/connect/refresh

# Google OAuth
GOOGLE_CLIENT_ID=....apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=GOCSPX-...
GOOGLE_REDIRECT_URI=https://api.example.com/auth/google/callback
```

### 3.3 起動後の確認

```bash
curl -s https://api.example.com/health
# → {"status":"ok"} 相当
```

### 3.4 Stripe Webhook

Stripe Dashboard で Webhook エンドポイントを登録:

```text
https://api.example.com/stripe/webhook
```

`STRIPE_WEBHOOK_SECRET` に表示されたシークレットを設定します。

---

## 4. Vercel（フロントエンド）

### 4.1 プロジェクト作成

1. [Vercel](https://vercel.com/) で GitHub リポジトリを Import
2. **Root Directory**: `frontend`
3. **Framework Preset**: Vite
4. **Build Command**: `npm run build`
5. **Output Directory**: `dist`

### 4.2 環境変数（Vercel）

| 変数 | 値 |
|------|-----|
| `VITE_API_BASE_URL` | `https://api.example.com`（末尾スラッシュなし） |
| `VITE_STRIPE_PUBLISHABLE_KEY` | `pk_live_...`（本番 Stripe 使用時） |

`VITE_*` は **ビルド時** に埋め込まれます。変更したら再デプロイが必要です。

### 4.3 SPA ルーティング（vercel.json）

`frontend/vercel.json` を置く場合の例:

```json
{
  "rewrites": [
    { "source": "/((?!assets/).*)", "destination": "/index.html" }
  ]
}
```

React Router の `/signin` `/projects/:id` 等が 404 にならないようにします。

### 4.4 カスタムドメイン

1. Vercel → Project → Domains → `app.example.com` を追加
2. DNS（CNAME）を Vercel の指示どおり設定
3. `PUBLIC_APP_URL` / `CORS_ALLOW_ORIGINS` を同じ URL に更新して API を再デプロイ

---

## 5. Google OAuth の本番設定

[docs/GOOGLE_OAUTH_SETUP.md](../GOOGLE_OAUTH_SETUP.md) に加え、Console で以下を登録します。

| 項目 | 値 |
|------|-----|
| 承認済み JavaScript 生成元 | `https://app.example.com` |
| 承認済みリダイレクト URI | `https://api.example.com/auth/google/callback` |

バックエンドの `GOOGLE_REDIRECT_URI` と **完全一致** させてください。

---

## 6. デプロイ順序（推奨）

```text
1. Neon 作成 → DATABASE_URL 取得
2. Upstash Redis 作成
3. API をデプロイ → migration 自動適用 → /health 確認
4. Vercel にフロントデプロイ（VITE_API_BASE_URL 設定）
5. CORS / PUBLIC_APP_URL / Cookie を揃えて API 再デプロイ
6. Google OAuth・Stripe Webhook を本番 URL で設定
7. 本番で購入者登録 → ログイン → 決済のスモークテスト
```

---

## 7. トラブルシューティング

| 症状 | 確認 |
|------|------|
| フロントは開くが API が失敗 | `VITE_API_BASE_URL`、API の `/health`、CORS |
| Google ログイン後に失敗 | `GOOGLE_REDIRECT_URI`、Redis、`redirect_uri_mismatch` |
| ログイン直後にログアウト扱い | `COOKIE_SECURE=true`、HTTPS、`CORS_ALLOW_ORIGINS` にフロント origin |
| WebSocket 接続失敗 | API が WS をサポートするホストか、Origin が CORS に含まれるか |
| 画像が消える | 永続ボリューム未設定 → 将来オブジェクトストレージ化を検討 |

---

## 8. セキュリティチェックリスト

- [ ] `.env` / 秘密値を Git に含めない
- [ ] 本番の `JWT_SECRET` 等はローカルと別値
- [ ] Neon の DB パスワードは Vercel には載せない（API 側のみ）
- [ ] `SEED_TEST_USER_*` は本番で無効
- [ ] Stripe は `STRIPE_USE_STUB=0`

---

## 参考: ローカルとの違い

| 項目 | ローカル（docker compose） | 本番 |
|------|---------------------------|------|
| フロント | nginx :80 | Vercel |
| API プロキシ | nginx が同一オリジン | `VITE_API_BASE_URL` で別オリジン |
| DB | Docker postgres | Neon |
| Redis | Docker redis | Upstash 等 |

ローカルでは `VITE_API_BASE_URL` を空にすると同一オリジン運用できますが、Vercel 本番では **必ず API の URL を設定**してください。
