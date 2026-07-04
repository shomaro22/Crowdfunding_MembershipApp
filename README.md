# Crowdfunding Membership App

クラウドファンディング型メンバーシップアプリ（React + Go）。API 契約はリポジトリ直下の [`openapi.yaml`](openapi.yaml) を正本とする。

## デモアカウント

| ロール | メール | パスワード |
|--------|--------|------------|
| 購入者 | `buyer@example.com` | `password123` |
| 出品者 | `seller@example.com` | `password123` |
| 管理者 | `admin@example.com` | `password123` |

出品者・管理者はログイン後に 2FA（TOTP）が必要です。シード実行時のコンソールに表示される 6 桁コードを入力してください。

## Stripe テストカード（デモ / サンドボックス）

`STRIPE_USE_STUB=1`（デフォルト）のときは決済画面のボタンでシミュレートできます。実 Stripe を使う場合は以下を入力します。

| 項目 | 値 |
|------|-----|
| カード番号 | `4242 4242 4242 4242` |
| 有効期限 | 未来の任意の月/年（例: `12/34`） |
| CVC | 任意の 3 桁（例: `123`） |

3D Secure テスト: `4000 0027 6000 3184`

## Google ログイン

1. ログイン / 新規登録画面の「Google でログイン」をクリック
2. Google アカウントを選択
3. **既存のメール登録アカウントがある場合** → 連携確認画面で「連携する」を押す
4. **新規の場合** → そのままログイン完了

設定手順: [docs/GOOGLE_OAUTH_SETUP.md](docs/GOOGLE_OAUTH_SETUP.md)  
フロー詳細: [docs/architecture/google-oauth-oidc.md](docs/architecture/google-oauth-oidc.md)

## 起動

Docker はリポジトリ直下の [`Dockerfile`](Dockerfile) のみを使用する（`docker compose` / CI の `backend-runtime` ターゲット）。

```bash
docker compose up -d --build
```

シード再投入（デモデータ修復含む）:

```bash
docker compose run --rm backend-seed dev -team -reset
```

Render 本番 API は [`Dockerfile.render`](Dockerfile.render) を参照。

## データベース ER 図

| ファイル | 用途 |
|----------|------|
| [`docs/database/schema.dbml`](docs/database/schema.dbml) | ER 図ソース（dbdiagram.io 用） |
| `backend/db/migrations/*.up.sql` | 実装の正本（矛盾時は migration を優先） |
| [`dbdiagram.pdf`](dbdiagram.pdf) | エクスポート済み PDF |

PDF の更新: [dbdiagram.io](https://dbdiagram.io/) で `schema.dbml` を Import → Export to PDF → ルートの `dbdiagram.pdf` を上書き。
