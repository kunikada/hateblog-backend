#!/usr/bin/env sh
set -eu

# Clear Redis cache for manual testing
# Run this after re-seeding the database

REDIS_HOST=${REDIS_HOST:-redis}
REDIS_PORT=${REDIS_PORT:-6379}

echo "==> Clearing Redis cache..."
echo "  Redis: ${REDIS_HOST}:${REDIS_PORT}"

if ! command -v redis-cli > /dev/null 2>&1; then
  echo "Error: redis-cli command not found. Please install redis-tools." >&2
  exit 1
fi

redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" FLUSHDB
echo "✓ Cache cleared successfully"
