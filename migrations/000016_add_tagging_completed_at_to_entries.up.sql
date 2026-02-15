ALTER TABLE entries
ADD COLUMN IF NOT EXISTS tagging_completed_at TIMESTAMP WITH TIME ZONE;

COMMENT ON COLUMN entries.tagging_completed_at IS 'Yahooタグ付け処理の完了日時（タグ0件を含む）';

UPDATE entries e
SET tagging_completed_at = e.created_at;

CREATE INDEX IF NOT EXISTS idx_entries_tagging_pending_created_at
ON entries (created_at DESC)
WHERE tagging_completed_at IS NULL;
