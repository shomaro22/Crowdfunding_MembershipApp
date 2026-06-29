# FanFunding Frontend (Phase 7)

Vite + React + React Router。Go バックエンド API 向け SPA。

## セットアップ

```bash
cd frontend
cp .env.example .env
npm install
npm run dev
```

- フロント: http://localhost:5173
- API: `VITE_API_BASE_URL`（既定 `http://localhost:8080`）

## セキュリティ方針

- **Access JWT はメモリのみ**（`src/lib/token-store.ts`）。localStorage / sessionStorage 不使用
- Refresh Token は HttpOnly Cookie（`credentials: 'include'`）
- リダイレクトは相対パスのみ（`sanitizeAppPath`）
- OAuth / Checkout URL にトークンを載せない
- 入力サニタイズ・パスワード強度チェック（NewFF 流用）
- `npm run check-no-any` で `any` 禁止

## デザイン

[NewFF_App](/Users/yamawakishoma/Desktop/NewFF_App) の FanFunding デザイン（紫テーマ、`.btn-primary` / `.card` 等）を流用。

## スクリプト

| コマンド | 内容 |
|---------|------|
| `npm run dev` | 開発サーバー |
| `npm run build` | 本番ビルド |
| `npm run lint` | ESLint + any チェック |
