-- Optimize indexes for better query performance
-- Based on Supabase Postgres Best Practices
--
-- Note: Partial index thresholds are based on actual API parameters:
--   - minUsers (bookmark_count): 5, 10, 50, 100, 500, 1000 (default: 5)
--   - Most commonly used value: 5 (default)

-- 1. created_atでのソート最適化(SortRecentパターン用)
CREATE INDEX idx_entries_created_desc ON entries (created_at DESC)
  WHERE created_at IS NOT NULL;

-- 2. bookmark_countでフィルタ + created_atでソートのパターン用複合インデックス
CREATE INDEX idx_entries_bookmark_created ON entries (bookmark_count, created_at DESC);

-- 3. 人気エントリー専用の部分インデックス(API minUsers=5, デフォルト値で最も使用頻度が高い)
CREATE INDEX idx_entries_min5_created ON entries (created_at DESC)
  WHERE bookmark_count >= 5;

-- 4. ホットソート用の部分インデックス(API minUsers=5以上)
CREATE INDEX idx_entries_hot_partial ON entries (bookmark_count DESC, created_at DESC)
  WHERE bookmark_count >= 5;

-- 5. entry_tagsのカバリングインデックス(loadTags最適化)
CREATE INDEX idx_entry_tags_entry_covering ON entry_tags (entry_id)
  INCLUDE (tag_id, score);

-- Add comments
COMMENT ON INDEX idx_entries_created_desc IS 'created_atでのソート最適化(降順)';
COMMENT ON INDEX idx_entries_bookmark_created IS 'bookmark_countフィルタ + created_atソート用の複合インデックス';
COMMENT ON INDEX idx_entries_min5_created IS '人気エントリー(5件以上)の部分インデックス - APIデフォルト値、最も使用頻度が高い';
COMMENT ON INDEX idx_entries_hot_partial IS 'ホットソート用の部分インデックス(5件以上)';
COMMENT ON INDEX idx_entry_tags_entry_covering IS 'loadTags最適化用のカバリングインデックス';
