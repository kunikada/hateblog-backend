-- Drop full-text search indexes

-- Reset statistics
ALTER TABLE entries ALTER COLUMN bookmark_count SET STATISTICS -1;

-- Drop indexes
DROP INDEX IF EXISTS idx_entries_excerpt_gin;
DROP INDEX IF EXISTS idx_entries_title_gin;
