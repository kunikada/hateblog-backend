#!/usr/bin/env bash
set -euo pipefail

# PostgreSQL pg_dump フルバックアップを gzip 圧縮し、S3 へアップロードする。
# 前提: docker compose で postgres サービスが稼働、AWS CLI が利用可能。

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

: "${POSTGRES_PASSWORD:?POSTGRES_PASSWORD を設定してください}"
: "${S3_BUCKET:?S3_BUCKET を設定してください}"

POSTGRES_DB="${POSTGRES_DB:-hateblog}"
POSTGRES_USER="${POSTGRES_USER:-hateblog}"
S3_PREFIX="${S3_PREFIX:-db-backups/}"
AWS_PROFILE="${AWS_PROFILE:-}"

TIMESTAMP="$(date +%Y%m%d_%H%M%S)"
OBJECT_KEY="${S3_PREFIX}backup_${TIMESTAMP}.sql.gz"

echo "[info] starting backup: db=${POSTGRES_DB}, s3://$S3_BUCKET/$OBJECT_KEY"

export PGPASSWORD="$POSTGRES_PASSWORD"

if [ -n "$AWS_PROFILE" ]; then
  docker compose exec -T postgres pg_dump -U "$POSTGRES_USER" "$POSTGRES_DB" \
    | gzip \
    | aws s3 cp - "s3://$S3_BUCKET/$OBJECT_KEY" --profile "$AWS_PROFILE"
else
  docker compose exec -T postgres pg_dump -U "$POSTGRES_USER" "$POSTGRES_DB" \
    | gzip \
    | aws s3 cp - "s3://$S3_BUCKET/$OBJECT_KEY"
fi

echo "[info] backup completed: s3://$S3_BUCKET/$OBJECT_KEY"
