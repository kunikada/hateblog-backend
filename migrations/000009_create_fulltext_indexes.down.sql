-- Drop full-text search index

-- Reset statistics
ALTER TABLE entries ALTER COLUMN bookmark_count SET STATISTICS -1;

-- Drop indexes
DROP INDEX IF EXISTS idx_entries_search_text_gin;
