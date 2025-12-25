#!/usr/bin/env sh
set -eu

# Manual test data seeding script for hateblog backend
# Prerequisites: PostgreSQL must be running and migrations must be applied
# Use this script to inject test data after starting the application

SQL_FILE=${SQL_FILE:-migrations/seeds/manual_api_seed.sql}
DB_HOST=${DB_HOST:-postgres}
DB_PORT=${DB_PORT:-5432}
DB_USER=${DB_USER:-hateblog}
DB_PASSWORD=${DB_PASSWORD:-changeme}
DB_NAME=${DB_NAME:-hateblog}
REDIS_HOST=${REDIS_HOST:-redis}
REDIS_PORT=${REDIS_PORT:-6379}

if [ ! -f "$SQL_FILE" ]; then
  echo "Error: Seed SQL file not found: $SQL_FILE" >&2
  exit 1
fi

echo "==> Seeding test data"
echo "  PostgreSQL: ${DB_HOST}:${DB_PORT}"
echo "  Database: ${DB_NAME}"
echo ""

# Check if psql is available
if ! command -v psql > /dev/null 2>&1; then
  echo "Error: psql command not found. Please install postgresql-client." >&2
  exit 1
fi

# Verify database connection
echo "==> Verifying database connection..."
if ! PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1" > /dev/null 2>&1; then
  echo "Error: Failed to connect to PostgreSQL at ${DB_HOST}:${DB_PORT}" >&2
  echo ""
  echo "Troubleshooting:" >&2
  echo "  - Check if postgres container is running: docker compose ps" >&2
  echo "  - Check if migrations were applied: docker compose logs postgres" >&2
  echo "  - Verify credentials in .env file" >&2
  exit 1
fi
echo "✓ Database connection successful"

# Verify that migrations have been applied (check for entries table)
echo "==> Checking database schema..."
table_count=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "
  SELECT COUNT(*)
  FROM information_schema.tables
  WHERE table_schema = 'public' AND table_name = 'entries'
" 2>/dev/null || echo "0")

if [ "$table_count" -eq 0 ]; then
  echo "Error: Database schema not initialized. Migrations may not have been applied." >&2
  echo "Solution: Restart the application container to run migrations automatically." >&2
  exit 1
fi
echo "✓ Database schema verified"

# Import seed data
echo ""
echo "==> Importing test data..."
if PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$SQL_FILE" > /dev/null 2>&1; then
  echo "✓ Test data imported successfully"
else
  echo "Warning: Test data import completed with errors. Check the database state." >&2
  exit 1
fi

# Clear Redis cache
echo ""
echo "==> Clearing Redis cache..."
if command -v redis-cli > /dev/null 2>&1; then
  if redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" FLUSHDB > /dev/null 2>&1; then
    echo "✓ Cache cleared successfully"
  else
    echo "Warning: Failed to clear Redis cache. You may see stale data." >&2
  fi
else
  # Try using docker exec if redis-cli is not available
  if command -v docker > /dev/null 2>&1; then
    container_name=$(docker ps --filter "ancestor=redis" --format "{{.Names}}" | head -n 1)
    if [ -n "$container_name" ] && docker exec "$container_name" redis-cli FLUSHDB > /dev/null 2>&1; then
      echo "✓ Cache cleared successfully (via docker)"
    else
      echo "Warning: Could not clear Redis cache. You may see stale data." >&2
      echo "  To clear manually: docker exec <redis-container> redis-cli FLUSHDB" >&2
    fi
  else
    echo "Warning: redis-cli not found. Cache not cleared." >&2
    echo "  To clear manually: redis-cli -h $REDIS_HOST -p $REDIS_PORT FLUSHDB" >&2
  fi
fi
