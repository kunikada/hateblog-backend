-- Create search_history table
CREATE TABLE IF NOT EXISTS search_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    query TEXT NOT NULL,
    searched_at DATE NOT NULL,
    count INTEGER NOT NULL DEFAULT 0,

    -- Constraints
    UNIQUE (query, searched_at),
    CONSTRAINT search_history_count_check CHECK (count >= 0)
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_search_history_searched_at ON search_history (searched_at DESC);
CREATE INDEX IF NOT EXISTS idx_search_history_count ON search_history (count DESC);

-- Add comment
COMMENT ON TABLE search_history IS '検索キーワードの日別集計テーブル';
COMMENT ON COLUMN search_history.id IS '集計ID（主キー）';
COMMENT ON COLUMN search_history.query IS '検索クエリ';
COMMENT ON COLUMN search_history.searched_at IS '検索日（日別集計）';
COMMENT ON COLUMN search_history.count IS '検索回数';
