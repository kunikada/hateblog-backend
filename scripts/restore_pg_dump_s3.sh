#!/usr/bin/env bash
set -euo pipefail

# S3 上の pg_dump(gzip) を取得し、PostgreSQL に復元する。
# 前提: docker compose で postgres サービスが稼働、AWS CLI が利用可能。

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

# .env があれば読み込む（POSTGRES_PASSWORD / DB_NAME / DB_USER 等を利用）
if [ -f .env ]; then
  set -a
  # shellcheck disable=SC1091
  . ./.env
  set +a
fi

: "${POSTGRES_PASSWORD:?POSTGRES_PASSWORD を設定してください}"
: "${S3_BUCKET:?S3_BUCKET を設定してください}"

DB_NAME="${DB_NAME:-hateblog}"
DB_USER="${DB_USER:-hateblog}"
S3_PREFIX="${S3_PREFIX:-db-backups/}"
AWS_PROFILE="${AWS_PROFILE:-}"

if [ -n "${S3_KEY:-}" ]; then
  OBJECT_KEY="$S3_KEY"
elif [ -n "${BACKUP_NAME:-}" ]; then
  OBJECT_KEY="${S3_PREFIX}${BACKUP_NAME}"
else
  echo "[error] S3_KEY か BACKUP_NAME を指定してください" >&2
  exit 1
fi

echo "[info] starting restore: db=${DB_NAME}, s3://$S3_BUCKET/$OBJECT_KEY"

export PGPASSWORD="$POSTGRES_PASSWORD"

if [ -n "$AWS_PROFILE" ]; then
  aws s3 cp "s3://$S3_BUCKET/$OBJECT_KEY" - --profile "$AWS_PROFILE" \
    | gunzip \
    | docker compose exec -T postgres psql -U "$DB_USER" "$DB_NAME"
else
  aws s3 cp "s3://$S3_BUCKET/$OBJECT_KEY" - \
    | gunzip \
    | docker compose exec -T postgres psql -U "$DB_USER" "$DB_NAME"
fi

echo "[info] restore completed: s3://$S3_BUCKET/$OBJECT_KEY"
