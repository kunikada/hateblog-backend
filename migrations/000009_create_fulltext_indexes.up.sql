-- Create full-text search GIN indexes using pg_bigm

-- Index for entries.title
CREATE INDEX idx_entries_title_gin ON entries USING gin (title gin_bigm_ops);

-- Index for entries.excerpt
CREATE INDEX idx_entries_excerpt_gin ON entries USING gin (excerpt gin_bigm_ops);

-- Adjust statistics for better query planning
ALTER TABLE entries ALTER COLUMN bookmark_count SET STATISTICS 1000;

-- Add comment
COMMENT ON INDEX idx_entries_title_gin IS 'タイトルの全文検索用GINインデックス（pg_bigm）';
COMMENT ON INDEX idx_entries_excerpt_gin IS '抜粋の全文検索用GINインデックス（pg_bigm）';
