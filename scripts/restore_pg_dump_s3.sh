#!/usr/bin/env bash
set -euo pipefail

# S3 上の pg_dump(gzip) を取得し、PostgreSQL に復元する。
# 前提: docker compose で postgres サービスが稼働、AWS CLI が利用可能。

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

: "${POSTGRES_PASSWORD:?POSTGRES_PASSWORD を設定してください}"
: "${S3_BUCKET:?S3_BUCKET を設定してください}"

POSTGRES_DB="${POSTGRES_DB:-hateblog}"
POSTGRES_USER="${POSTGRES_USER:-hateblog}"
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

echo "[info] starting restore: db=${POSTGRES_DB}, s3://$S3_BUCKET/$OBJECT_KEY"

export PGPASSWORD="$POSTGRES_PASSWORD"

if [ -n "$AWS_PROFILE" ]; then
  aws s3 cp "s3://$S3_BUCKET/$OBJECT_KEY" - --profile "$AWS_PROFILE" \
    | gunzip \
    | docker compose exec -T postgres psql -U "$POSTGRES_USER" "$POSTGRES_DB"
else
  aws s3 cp "s3://$S3_BUCKET/$OBJECT_KEY" - \
    | gunzip \
    | docker compose exec -T postgres psql -U "$POSTGRES_USER" "$POSTGRES_DB"
fi

echo "[info] restore completed: s3://$S3_BUCKET/$OBJECT_KEY"
