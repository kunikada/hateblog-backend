-- Add index for hot lists and rankings based on created_at
CREATE INDEX IF NOT EXISTS idx_entries_bookmark_count_created_at
    ON entries (bookmark_count DESC, created_at DESC);

-- Rebuild archive_counts using created_at day
TRUNCATE TABLE archive_counts;

INSERT INTO archive_counts (day, threshold, count)
SELECT DATE(created_at) AS day, t.threshold, COUNT(1)
FROM entries
CROSS JOIN (VALUES (5), (10), (50), (100), (500), (1000)) AS t(threshold)
WHERE entries.bookmark_count >= t.threshold
GROUP BY day, t.threshold;
