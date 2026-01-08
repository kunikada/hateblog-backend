-- Create full-text search GIN index for entries.url using pg_bigm
CREATE INDEX idx_entries_url_gin ON entries USING gin (url gin_bigm_ops);

COMMENT ON INDEX idx_entries_url_gin IS 'URLの全文検索用GINインデックス（pg_bigm）';
