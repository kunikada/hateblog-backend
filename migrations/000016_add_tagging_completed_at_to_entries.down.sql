DROP INDEX IF EXISTS idx_entries_tagging_pending_created_at;

ALTER TABLE entries
DROP COLUMN IF EXISTS tagging_completed_at;
