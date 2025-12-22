-- Enable pg_bigm extension for full-text search
CREATE EXTENSION IF NOT EXISTS pg_bigm;

-- Add comment
COMMENT ON EXTENSION pg_bigm IS '日本語対応のバイグラムベース全文検索拡張';
