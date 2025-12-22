#!/usr/bin/env sh
set -eu

# 手動APIテスト用シード投入スクリプト
# 依存: docker compose, psql (コンテナ内)

SQL_FILE=${SQL_FILE:-migrations/seeds/manual_api_seed.sql}
COMPOSE_CMD=${COMPOSE_CMD:-docker compose}
POSTGRES_SERVICE=${POSTGRES_SERVICE:-postgres}
DB_USER=${DB_USER:-hateblog}
DB_PASSWORD=${DB_PASSWORD:-changeme}
DB_NAME=${DB_NAME:-hateblog}

if [ ! -f "$SQL_FILE" ]; then
  echo "シードSQLが見つかりません: $SQL_FILE" >&2
  exit 1
fi

echo "==> Postgresコンテナの起動確認 (${POSTGRES_SERVICE})"
if ! $COMPOSE_CMD ps --services --filter "status=running" | grep -q "^${POSTGRES_SERVICE}$"; then
  echo "Postgresが起動していません。先に 'docker compose up -d' を実行してください。" >&2
  exit 1
fi

echo "==> シード投入開始 (${SQL_FILE})"
$COMPOSE_CMD exec -T -e PGPASSWORD="$DB_PASSWORD" "$POSTGRES_SERVICE" \
  psql -U "$DB_USER" -d "$DB_NAME" -f "$SQL_FILE"

echo "✓ シード投入が完了しました"
