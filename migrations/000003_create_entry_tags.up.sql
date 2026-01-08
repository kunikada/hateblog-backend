-- Create entry_tags table
CREATE TABLE IF NOT EXISTS entry_tags (
    entry_id UUID NOT NULL,
    tag_id UUID NOT NULL,
    score INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    PRIMARY KEY (entry_id, tag_id),
    FOREIGN KEY (entry_id) REFERENCES entries(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE,
    CONSTRAINT entry_tags_score_check CHECK (score >= 0 AND score <= 100)
);

-- Create indexes
CREATE INDEX idx_entry_tags_tag_entry ON entry_tags (tag_id, entry_id);
CREATE INDEX idx_entry_tags_score ON entry_tags (entry_id, score DESC);

-- Add comment
COMMENT ON TABLE entry_tags IS 'エントリーとタグの多対多リレーションを管理する中間テーブル。スコア付き';
COMMENT ON COLUMN entry_tags.entry_id IS 'エントリーID（外部キー）';
COMMENT ON COLUMN entry_tags.tag_id IS 'タグID（外部キー）';
COMMENT ON COLUMN entry_tags.score IS 'タグのスコア（Yahoo! APIから取得した重要度、0〜100）';
COMMENT ON COLUMN entry_tags.created_at IS 'レコード作成日時';
