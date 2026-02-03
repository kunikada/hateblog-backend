DROP INDEX IF EXISTS idx_entries_bookmark_count_created_at;

TRUNCATE TABLE archive_counts;

INSERT INTO archive_counts (day, threshold, count)
SELECT DATE(posted_at) AS day, t.threshold, COUNT(1)
FROM entries
CROSS JOIN (VALUES (5), (10), (50), (100), (500), (1000)) AS t(threshold)
WHERE entries.bookmark_count >= t.threshold
GROUP BY day, t.threshold;
