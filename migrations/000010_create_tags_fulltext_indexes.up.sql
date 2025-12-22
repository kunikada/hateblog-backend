-- Create full-text search GIN index for tags.name using pg_bigm

CREATE INDEX IF NOT EXISTS idx_tags_name_gin ON tags USING gin (name gin_bigm_ops);

COMMENT ON INDEX idx_tags_name_gin IS 'タグ名の全文検索用GINインデックス（pg_bigm）';

