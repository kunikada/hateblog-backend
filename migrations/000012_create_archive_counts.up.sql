-- Create archive_counts table
CREATE TABLE IF NOT EXISTS archive_counts (
    day DATE NOT NULL,
    bookmark_count INTEGER NOT NULL,
    count INTEGER NOT NULL,

    -- Constraints
    PRIMARY KEY (day, bookmark_count),
    CONSTRAINT archive_counts_bookmark_count_check CHECK (bookmark_count >= 0),
    CONSTRAINT archive_counts_count_check CHECK (count >= 0)
);

-- Create indexes
CREATE INDEX idx_archive_counts_bookmark_day ON archive_counts (bookmark_count, day DESC);

-- Backfill aggregated counts
INSERT INTO archive_counts (day, bookmark_count, count)
SELECT DATE(posted_at) AS day, bookmark_count, COUNT(1)
FROM entries
GROUP BY day, bookmark_count;

-- Add comment
COMMENT ON TABLE archive_counts IS '日別エントリー数の事前集計テーブル';
COMMENT ON COLUMN archive_counts.day IS '日付（YYYY-MM-DD）';
COMMENT ON COLUMN archive_counts.bookmark_count IS 'ブックマーク件数';
COMMENT ON COLUMN archive_counts.count IS '件数';
