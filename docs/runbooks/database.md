# Runbook: データベース障害

## 症状

- `/health` が 503
- API が `database connection` エラー

## 初動

1. `docker compose ps postgres` でコンテナ状態確認
2. `docker compose logs postgres --tail=100` でエラー確認
3. ディスク容量・接続数上限を確認

## 復旧

```bash
docker compose restart postgres
docker compose up -d backend
```

## バックアップ

```bash
docker compose exec postgres pg_dump -U postgres crowdfund > backup-$(date +%Y%m%d).sql
```

## リストア

```bash
cat backup-YYYYMMDD.sql | docker compose exec -T postgres psql -U postgres crowdfund
```
