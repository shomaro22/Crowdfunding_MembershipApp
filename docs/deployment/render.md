# Render で API をデプロイする手順

Neon（PostgreSQL）+ Render（Go API）+ Vercel（フロント）の構成です。

---

## 前提

- Neon の `DATABASE_URL` は取得済み（ローカル `go run .` で `/health` が `db: ok`）
- GitHub にリポジトリが push 済み
- Render アカウント作成済み

---

## 1. Redis（Render Key Value 推奨）

OAuth 状態・レート制限に Redis が必須です。

### 方法 A: Web Service にリンク（いちばん簡単）

1. [Render Dashboard](https://dashboard.render.com/) → **New +** → **Key Value**
2. 名前例: `crowdfund-redis`、**リージョンは API の Web Service と同じ**
3. 作成後、**API の Web Service** → **Environment**
4. **Add Environment Variable** → **Add from Render service**（または Key Value を選択）
5. 作成した Key Value を選ぶ → **`REDIS_URL` が自動追加**される
6. Save → 再デプロイ

アプリは `REDIS_URL` を読み取ります（`REDIS_ADDR` は不要）。

### 方法 B: 手動で環境変数を入れる

Key Value → **Connections** → **Internal Redis URL** をコピー:

```text
redis://red:xxxxxxxx@red-xxxxx:6379
```

Web Service の Environment に **どちらか一方**:

| 方式 | 設定 |
|------|------|
| 推奨 | `REDIS_URL` = 上記 URL をそのまま貼る |
| 分割 | `REDIS_ADDR` = `red-xxxxx:6379`、`REDIS_PASSWORD` = URL 内のパスワード |

### Upstash を使う場合

Upstash の **Redis URL**（`rediss://...`）を `REDIS_URL` に設定。TLS 付き URL に対応済みです。

### 動作確認

```bash
curl -s https://<your-service>.onrender.com/health
# "redis":"ok" であること
```

---

## 2. Web Service 作成

1. **New +** → **Web Service**
2. GitHub リポジトリを接続
3. 設定:

| 項目 | 値 |
|------|-----|
| Name | `crowdfund-api` 等 |
| Region | Neon に近いリージョン（例: Singapore） |
| Branch | デプロイするブランチ |
| Runtime | **Docker** |
| Dockerfile Path | `./Dockerfile.render`（API 専用。`./Dockerfile` だと nginx が選ばれる） |
| Docker Build Context | `.`（リポジトリルート） |
| Docker Target | UI にある場合のみ `backend-runtime`（`Dockerfile.render` なら不要） |

4. **Instance Type**: Free はスリープあり。本番は Starter 以上を推奨

---

## 3. 環境変数

Render → Web Service → **Environment** に追加。

### 必須

```env
APP_ENV=production

DATABASE_URL=postgresql://USER:PASSWORD@HOST/neondb?sslmode=require

REDIS_ADDR=red-xxxxx:6379
REDIS_PASSWORD=xxxxxxxx

JWT_SECRET=（32文字以上・ローカルと別のランダム値）
REFRESH_SECRET=（32文字以上）
TOTP_ENCRYPTION_KEY=（32文字以上）
KYC_SIGNING_KEY=（32文字以上）
PENDING_2FA_KEY=（32文字以上）

COOKIE_SECURE=true
TOTP_ISSUER=CrowdfundApp

PUBLIC_ROOT=/app/data/public
PRIVATE_ROOT=/app/data/private
```

### URL（初回デプロイ後に確定 URL で更新）

初回は Render が `https://crowdfund-api.onrender.com` のような URL を付与します。**一度デプロイしてから** 以下を設定し、**Manual Deploy** で再デプロイします。

```env
API_PUBLIC_URL=https://crowdfund-api.onrender.com
PUBLIC_APP_URL=https://your-app.vercel.app
CORS_ALLOW_ORIGINS=https://your-app.vercel.app
```

`PORT` は **Render が自動注入**するため、手動で `8080` を設定しないでください。

### Google OAuth（使う場合）

```env
GOOGLE_CLIENT_ID=....apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=GOCSPX-...
GOOGLE_REDIRECT_URI=https://crowdfund-api.onrender.com/auth/google/callback
```

### Stripe（使う場合）

```env
STRIPE_USE_STUB=0
STRIPE_SECRET_KEY=sk_test_...
STRIPE_WEBHOOK_SECRET=whsec_...
STRIPE_CONNECT_RETURN_URL=https://your-app.vercel.app/seller/connect/return
STRIPE_CONNECT_REFRESH_URL=https://your-app.vercel.app/seller/connect/refresh
```

### 本番で設定しないもの

```env
SEED_TEST_USER_TEAM=
SEED_TEST_USER_RESET=
TEST_USER_PASSWORD=
```

---

## 4. デプロイと確認

1. **Create Web Service**（または Save Changes → Deploy）
2. ログで migration エラーがないか確認
3. ヘルスチェック:

```bash
curl -s https://crowdfund-api.onrender.com/health
```

期待:

```json
{"status":"ok","db":"ok","redis":"ok"}
```

4. `API_PUBLIC_URL` / `GOOGLE_REDIRECT_URI` を実 URL に更新して再デプロイ

---

## 5. Vercel（フロント）との接続

1. Vercel で `frontend/` をデプロイ
2. 環境変数: `VITE_API_BASE_URL=https://crowdfund-api.onrender.com`
3. Render 側を更新:

```env
PUBLIC_APP_URL=https://xxxx.vercel.app
CORS_ALLOW_ORIGINS=https://xxxx.vercel.app
```

4. Render を再デプロイ → ブラウザでログイン確認

---

## 6. オプション: Persistent Disk

アップロード画像・KYC 書類はコンテナの `/app/data` に保存されます。再起動で消えるため、本番では:

1. Web Service → **Disks** → Add Disk（マウントパス `/app/data`）
2. `PUBLIC_ROOT` / `PRIVATE_ROOT` はそのまま `/app/data/public` 等

---

## 7. render.yaml（任意）

リポジトリルートに `render.yaml` を置くと Blueprint から一括作成できます。手動作成でも問題ありません。

---

## トラブルシューティング

| 症状 | 対処 |
|------|------|
| Build failed: wrong target | Docker Target を `backend-runtime` に |
| `db` が ng | `DATABASE_URL`、Neon の IP 制限、`sslmode=require` |
| `redis` が ng | `REDIS_ADDR` / `REDIS_PASSWORD`、Key Value と同一リージョン |
| 502 / スリープ | Free プランのコールドスタート。Starter または定期 ping |
| CORS エラー | `CORS_ALLOW_ORIGINS` に Vercel の origin（末尾スラッシュなし） |
| Cookie が効かない | `COOKIE_SECURE=true`、HTTPS、`credentials: include` は API ドメインへ |

---

## 次のチェックリスト

- [ ] Render `/health` が ok
- [ ] Vercel デプロイ + `VITE_API_BASE_URL`
- [ ] `CORS_ALLOW_ORIGINS` / `PUBLIC_APP_URL` 更新
- [ ] Google OAuth リダイレクト URI 登録
- [ ] 購入者登録・ログインのスモークテスト
