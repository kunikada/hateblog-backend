-- Create archive_counts table
CREATE TABLE IF NOT EXISTS archive_counts (
    day DATE NOT NULL,
    threshold INTEGER NOT NULL,
    count INTEGER NOT NULL,

    -- Constraints
    PRIMARY KEY (day, threshold),
    CONSTRAINT archive_counts_threshold_check CHECK (threshold IN (5, 10, 50, 100, 500, 1000)),
    CONSTRAINT archive_counts_count_check CHECK (count >= 0)
);

-- Create indexes
CREATE INDEX idx_archive_counts_threshold_day ON archive_counts (threshold, day DESC);

-- Backfill aggregated counts
INSERT INTO archive_counts (day, threshold, count)
SELECT DATE(posted_at) AS day, t.threshold, COUNT(1)
FROM entries
CROSS JOIN (VALUES (5), (10), (50), (100), (500), (1000)) AS t(threshold)
WHERE entries.bookmark_count >= t.threshold
GROUP BY day, t.threshold;

-- Add comment
COMMENT ON TABLE archive_counts IS '日別エントリー数の事前集計テーブル';
COMMENT ON COLUMN archive_counts.day IS '日付（YYYY-MM-DD）';
COMMENT ON COLUMN archive_counts.threshold IS '閾値（5, 10, 50, 100, 500, 1000）';
COMMENT ON COLUMN archive_counts.count IS 'threshold以上の件数';
