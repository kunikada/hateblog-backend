-- Create full-text search GIN index using pg_bigm

-- Index for entries.search_text
CREATE INDEX IF NOT EXISTS idx_entries_search_text_gin ON entries USING gin (search_text gin_bigm_ops);

-- Adjust statistics for better query planning
ALTER TABLE entries ALTER COLUMN bookmark_count SET STATISTICS 1000;

-- Add comment
COMMENT ON INDEX idx_entries_search_text_gin IS '検索用結合テキストの全文検索用GINインデックス（pg_bigm）';
