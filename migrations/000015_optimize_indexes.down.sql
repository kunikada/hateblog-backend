-- Rollback index optimizations

DROP INDEX IF EXISTS idx_entry_tags_entry_covering;
DROP INDEX IF EXISTS idx_entries_hot_partial;
DROP INDEX IF EXISTS idx_entries_min5_created;
DROP INDEX IF EXISTS idx_entries_bookmark_created;
DROP INDEX IF EXISTS idx_entries_created_desc;
