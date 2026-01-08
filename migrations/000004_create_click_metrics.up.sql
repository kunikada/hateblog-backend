-- Create click_metrics table
CREATE TABLE IF NOT EXISTS click_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entry_id UUID NOT NULL,
    clicked_at DATE NOT NULL,
    count INTEGER NOT NULL DEFAULT 0,

    -- Constraints
    UNIQUE (entry_id, clicked_at),
    FOREIGN KEY (entry_id) REFERENCES entries(id) ON DELETE CASCADE,
    CONSTRAINT click_metrics_count_check CHECK (count >= 0)
);

-- Create indexes
CREATE INDEX idx_click_metrics_clicked_at ON click_metrics (clicked_at DESC);
CREATE INDEX idx_click_metrics_clicked_entry ON click_metrics (clicked_at, entry_id);

-- Add comment
COMMENT ON TABLE click_metrics IS 'エントリーへのクリック数の日別集計テーブル';
COMMENT ON COLUMN click_metrics.id IS '集計ID（主キー）';
COMMENT ON COLUMN click_metrics.entry_id IS 'エントリーID（外部キー）';
COMMENT ON COLUMN click_metrics.clicked_at IS 'クリック日（日別集計）';
COMMENT ON COLUMN click_metrics.count IS 'クリック回数';
