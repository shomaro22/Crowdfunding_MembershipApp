# クラウドファンディング・メンバーシップアプリ

## 概要

本アプリケーションは、クリエイターが月額メンバーシッププラン（ブロンズ / シルバー / ゴールド）を提供し、支援者が Stripe で課金して会員特典・メッセージ機能を利用するクラウドファンディング型メンバーシップシステムです。

主な特徴:

- プロジェクトの作成・公開申請・管理者審査
- 3 段階メンバーシッププランと Stripe 決済（カード / コンビニ）
- WebSocket によるプロジェクト内メッセージのリアルタイム配信
- Google OAuth ログインと Credentials 認証
- 出品者・管理者向け TOTP 二要素認証（2FA）
- 管理画面（ユーザー管理・通報・活動ログ・バッチ状況）
- Redis ベースの Rate Limit によるセキュリティ対策

## 想定ユーザー

- **購入者（BUYER）** — プロジェクト閲覧・支援・メッセージ閲覧
- **出品者（SELLER）** — プロジェクト作成・KYC 登録・Stripe Connect・メッセージ投稿
- **管理者（ADMIN）** — 審査・ユーザー管理・セキュリティイベント確認

## 主なフロー

1. 購入者がアカウント登録（即時 ACTIVE）または Google ログイン
2. 公開済みプロジェクトを閲覧し、プランを選択して Stripe 決済
3. 支援者として WebSocket メッセージルームに参加
4. 出品者がプロジェクトを作成 → 管理者審査 → 公開
5. 出品者が KYC 完了後 Stripe Connect を接続し、売上を受け取る
6. 管理者が審査・通報対応・活動ログを確認

## 技術スタック

### Backend

- Go (Echo)
- PostgreSQL
- Redis（Rate Limit / セッション補助）
- GORM
- Stripe SDK（Connect / Checkout / Webhook）
- WebSocket Hub（`wshub`）

### Frontend

- React 19
- TypeScript
- Vite
- Tailwind CSS
- React Router
- Stripe Elements

### その他

- Docker / Docker Compose
- Playwright（E2E / `e2e/`）
- k6（負荷試験 / `k6/`）
- GitHub Actions（CI / Nightly）

## アーキテクチャ

バックエンドはレイヤ分離を意識した構成です。

- `model` — ドメイン型・enum・バリデーション
- `repository` — DB / Redis / ファイルストレージ
- `usecase` — 業務ロジック（View 返却）
- `controller` — HTTP JSON Wire DTO・Cookie 変換
- `wshub` — WebSocket プロトコル（usecase を import しない）
- `bootstrap` — 依存注入・View→WS DTO 変換
- `middleware` — 認証・Rate Limit・監査

認証は **RefreshToken**（httpOnly Cookie）+ **AuthToken**（クライアントメモリのみ）です。

## デプロイ先

- **Backend API (Render)**: `https://crowdfunding-membershipapp.onrender.com`
- **Health Check**: `https://crowdfunding-membershipapp.onrender.com/health`

補足:

- Render の Blueprint 例: [`render.yaml`](render.yaml)（`Dockerfile.render` を使用）
- フロントエンドを別ホスト（例: Vercel）に置く場合は `VITE_API_BASE_URL` に Render API の URL を設定し、Render 側の `CORS_ALLOW_ORIGINS` にフロントのオリジンを追加してください
- Google OAuth を使う場合は Google Cloud Console で `GOOGLE_REDIRECT_URI`（例: `https://<your-domain>/auth/google/callback`）を登録します

## 起動（全体）

Docker はリポジトリ直下の [`Dockerfile`](Dockerfile) のみを使用します（`docker compose` / CI の `backend-runtime` ターゲット）。

```bash
docker compose up -d --build
```

DB・Redis・バックエンド・フロントエンドを一括起動します。  
初回起動時は `backend-seed` が完了後に API が起動し、デモユーザーとサンプルプロジェクトを投入します。

| サービス                | URL / ポート            |
| ----------------------- | ----------------------- |
| フロントエンド（nginx） | `http://localhost`      |
| バックエンド API        | `http://localhost:8080` |
| PostgreSQL              | `localhost:5432`        |
| Redis                   | `localhost:6379`        |

シード再投入（デモデータ修復含む）:

```bash
docker compose run --rm backend-seed dev -team -reset
```

## バックエンド起動（個別）

```bash
docker compose up -d postgres redis backend-seed
cp .env.example .env   # ルートの .env.example を参照
cd backend
go run .
```

主な環境変数（ルート [`.env.example`](.env.example)）:

| 変数                                        | 説明                                                    |
| ------------------------------------------- | ------------------------------------------------------- |
| `DATABASE_URL`                              | PostgreSQL 接続文字列                                   |
| `REDIS_ADDR`                                | Redis アドレス（空だと Rate Limit 用 UC は無効）        |
| `JWT_SECRET` / `REFRESH_SECRET`             | 各 32 文字以上                                          |
| `TOTP_ENCRYPTION_KEY` / `PENDING_2FA_KEY`   | 2FA 用（各 32 文字以上）                                |
| `STRIPE_USE_STUB`                           | `1`（デフォルト）で Stripe スタブ決済                   |
| `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET` | Google OAuth（未設定時は OAuth 無効）                   |
| `RATE_LIMIT_DISABLED`                       | `1` で HTTP Rate Limit 無効（**CI k6 専用。本番禁止**） |

## フロントエンド起動（個別）

```bash
cd frontend
cp .env.example .env
npm ci
npm run dev
```

開発サーバー: `http://localhost:5173`（API は `VITE_API_BASE_URL=http://localhost:8080`）

## デモアカウント

`docker compose` 初回起動または seed 実行後に利用できます。

| ロール | メール               | パスワード    |
| ------ | -------------------- | ------------- |
| 購入者 | `buyer@example.com`  | `password123` |
| 出品者 | `seller@example.com` | `password123` |
| 管理者 | `admin@example.com`  | `password123` |

出品者・管理者はログイン後に **2FA（TOTP）** が必要です。seed 実行時のコンソールに表示される 6 桁コード、またはシード固定シークレット `JBSWY3DPEHPK3PXP`（管理者デモ用）から生成したコードを入力してください。

### seed コマンド

```bash
# Docker 経由（推奨）
docker compose run --rm backend-seed dev -team -reset

# ローカル go run
cd backend
go run ./cmd/seed dev              # buyer@example.com のみ
go run ./cmd/seed dev -team        # buyer / seller / admin
go run ./cmd/seed dev -team -reset # 既存ユーザーのパスワード・ロールを更新
```

## Stripe テストカード（デモ / サンドボックス）

`STRIPE_USE_STUB=1`（Docker デフォルト）のときは決済画面のボタンでシミュレートできます。実 Stripe を使う場合は以下を入力します。

| 項目       | 値                               |
| ---------- | -------------------------------- |
| カード番号 | `4242 4242 4242 4242`            |
| 有効期限   | 未来の任意の月/年（例: `12/34`） |
| CVC        | 任意の 3 桁（例: `123`）         |

3D Secure テスト: `4000 0027 6000 3184`

## Google ログイン

1. ログイン / 新規登録画面の「Google でログイン」をクリック
2. Google アカウントを選択
3. **既存のメール登録アカウントがある場合** → 連携確認画面で「連携する」を押す
4. **新規の場合** → そのままログイン完了

ローカル Docker では `.env` または `docker-compose.yml` の `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET` / `GOOGLE_REDIRECT_URI`（既定: `http://localhost/auth/google/callback`）を設定してください。未設定時は OAuth エンドポイントは無効応答（404 等）となります。

## 画像アップロードの制約

- API: `POST /media/images`
- 許可 MIME: `image/jpeg`, `image/png`, `image/webp`
- 最大サイズ: **10 MiB**
- エラー: 非対応は `415 Unsupported Media Type`、サイズ超過は `413 Payload Too Large`

## テスト

以下のテストを実施可能です。

- 静的解析（frontend lint / `go vet`）
- 単体・結合テスト（Go）
- E2E テスト（Playwright）
- API 負荷試験（k6）

### Backend テスト

```bash
cd backend
go vet ./...
go build ./...
go test ./...
go test -tags=integration -count=1 ./...
```

### Frontend lint / build

```bash
cd frontend
npm ci
npm run lint
npm run build
```

### E2E（Playwright）

スタック起動後、`e2e/` から実行します（CI と同じく `http://localhost` を向きます）。

```bash
docker compose up -d
cd e2e
npm ci
npx playwright install chromium
npm test
```

環境変数 `E2E_BASE_URL`（既定: `http://localhost`）で向き先を変更できます。

### k6

CI / Nightly では `RATE_LIMIT_DISABLED=1` でスタックを起動し、OAuth 等のスパイク試験が Rate Limit で 429 にならないようにしています。

```bash
docker compose up -d
# ローカルで OAuth スパイク等を再現する場合
RATE_LIMIT_DISABLED=1 docker compose up -d --force-recreate backend

mkdir -p k6/reports
k6 run k6/smoke.js
k6 run k6/project-list.js
k6 run k6/auth-load.js
k6 run k6/oauth-load.js
k6 run k6/ws-load.js
# 管理者 2FA
ADMIN_TOTP_CODE=$(node -e "const {authenticator}=require('otplib'); console.log(authenticator.generate('JBSWY3DPEHPK3PXP'))") \
  k6 run k6/admin-load.js
```

環境変数:

| 変数       | 既定               | 説明                              |
| ---------- | ------------------ | --------------------------------- |
| `BASE_URL` | `http://localhost` | API のベース URL                  |
| `K6_ENV`   | `development`      | `k6/lib/config.js` のプロファイル |

## CI（GitHub Actions）

[`.github/workflows/ci.yml`](.github/workflows/ci.yml) で主に以下を実行します。

| Job        | 内容                                                         |
| ---------- | ------------------------------------------------------------ |
| `backend`  | `go vet` / build / test / race / integration / coverage gate |
| `frontend` | lint / build                                                 |
| `docker`   | backend イメージビルド                                       |
| `e2e`      | `docker compose up` + Playwright                             |
| `k6`       | `RATE_LIMIT_DISABLED=1` でスタック起動 + k6 シナリオ         |

Nightly（[`.github/workflows/nightly.yml`](.github/workflows/nightly.yml)）では追加で `stress-suite.js` 等を実行します。

## API / 仕様

- OpenAPI 定義: [`openapi.yaml`](openapi.yaml)
- DB スキーマの正本: `backend/db/migrations/*.up.sql`
- ER 図（DBML）: [`docs/database/schema.dbml`](docs/database/schema.dbml) → [dbdiagram.io](https://dbdiagram.io) で Import し PDF を [`dbdiagram.pdf`](dbdiagram.pdf) に反映

## 注意点

- `STRIPE_USE_STUB=1` では本物の Stripe 課金は発生しません。
- `RATE_LIMIT_DISABLED=1` は **CI k6 専用** です。本番・ステージングでは設定しないでください。
- コマンドは実行ディレクトリ（ルート / `backend` / `frontend` / `e2e`）を正しく選んで実行してください。
