-- Initialize PostgreSQL extensions
-- This script runs automatically on container initialization via /docker-entrypoint-initdb.d/
-- It must run before migrations to ensure pg_bigm is available for migration 000008

CREATE EXTENSION IF NOT EXISTS pg_bigm;
