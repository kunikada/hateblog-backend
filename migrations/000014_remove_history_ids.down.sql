-- Restore id columns to history tables

-- click_metrics: restore id column
ALTER TABLE click_metrics DROP CONSTRAINT click_metrics_pkey;
ALTER TABLE click_metrics ADD COLUMN id UUID DEFAULT gen_random_uuid();
UPDATE click_metrics SET id = gen_random_uuid() WHERE id IS NULL;
ALTER TABLE click_metrics ALTER COLUMN id SET NOT NULL;
ALTER TABLE click_metrics ADD PRIMARY KEY (id);
ALTER TABLE click_metrics ADD CONSTRAINT click_metrics_entry_id_clicked_at_key UNIQUE (entry_id, clicked_at);
COMMENT ON COLUMN click_metrics.id IS '集計ID（主キー）';
COMMENT ON COLUMN click_metrics.entry_id IS 'エントリーID（外部キー）';
COMMENT ON COLUMN click_metrics.clicked_at IS 'クリック日（日別集計）';

-- tag_view_history: restore id column
ALTER TABLE tag_view_history DROP CONSTRAINT tag_view_history_pkey;
ALTER TABLE tag_view_history ADD COLUMN id UUID DEFAULT gen_random_uuid();
UPDATE tag_view_history SET id = gen_random_uuid() WHERE id IS NULL;
ALTER TABLE tag_view_history ALTER COLUMN id SET NOT NULL;
ALTER TABLE tag_view_history ADD PRIMARY KEY (id);
ALTER TABLE tag_view_history ADD CONSTRAINT tag_view_history_tag_id_viewed_at_key UNIQUE (tag_id, viewed_at);
COMMENT ON COLUMN tag_view_history.id IS '集計ID（主キー）';
COMMENT ON COLUMN tag_view_history.tag_id IS 'タグID（外部キー）';
COMMENT ON COLUMN tag_view_history.viewed_at IS '閲覧日（日別集計）';

-- search_history: restore id column
ALTER TABLE search_history DROP CONSTRAINT search_history_pkey;
ALTER TABLE search_history ADD COLUMN id UUID DEFAULT gen_random_uuid();
UPDATE search_history SET id = gen_random_uuid() WHERE id IS NULL;
ALTER TABLE search_history ALTER COLUMN id SET NOT NULL;
ALTER TABLE search_history ADD PRIMARY KEY (id);
ALTER TABLE search_history ADD CONSTRAINT search_history_query_searched_at_key UNIQUE (query, searched_at);
COMMENT ON COLUMN search_history.id IS '集計ID（主キー）';
COMMENT ON COLUMN search_history.query IS '検索クエリ';
COMMENT ON COLUMN search_history.searched_at IS '検索日（日別集計）';
