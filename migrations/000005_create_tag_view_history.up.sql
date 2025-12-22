-- Create tag_view_history table
CREATE TABLE IF NOT EXISTS tag_view_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tag_id UUID NOT NULL,
    viewed_at DATE NOT NULL,
    count INTEGER NOT NULL DEFAULT 0,

    -- Constraints
    UNIQUE (tag_id, viewed_at),
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE,
    CONSTRAINT tag_view_history_count_check CHECK (count >= 0)
);

-- Create indexes
CREATE INDEX idx_tag_view_history_viewed_at ON tag_view_history (viewed_at DESC);

-- Add comment
COMMENT ON TABLE tag_view_history IS 'タグ別エントリー一覧ページの閲覧数の日別集計テーブル';
COMMENT ON COLUMN tag_view_history.id IS '集計ID（主キー）';
COMMENT ON COLUMN tag_view_history.tag_id IS 'タグID（外部キー）';
COMMENT ON COLUMN tag_view_history.viewed_at IS '閲覧日（日別集計）';
COMMENT ON COLUMN tag_view_history.count IS '閲覧回数';
