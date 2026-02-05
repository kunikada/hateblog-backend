-- Remove id columns from history tables and use composite primary keys

-- click_metrics: (entry_id, clicked_at) -> PRIMARY KEY
ALTER TABLE click_metrics DROP CONSTRAINT click_metrics_entry_id_clicked_at_key;
ALTER TABLE click_metrics DROP COLUMN id;
ALTER TABLE click_metrics ADD PRIMARY KEY (entry_id, clicked_at);
COMMENT ON COLUMN click_metrics.entry_id IS 'エントリーID（複合主キー・外部キー）';
COMMENT ON COLUMN click_metrics.clicked_at IS 'クリック日（複合主キー・日別集計）';

-- tag_view_history: (tag_id, viewed_at) -> PRIMARY KEY
ALTER TABLE tag_view_history DROP CONSTRAINT tag_view_history_tag_id_viewed_at_key;
ALTER TABLE tag_view_history DROP COLUMN id;
ALTER TABLE tag_view_history ADD PRIMARY KEY (tag_id, viewed_at);
COMMENT ON COLUMN tag_view_history.tag_id IS 'タグID（複合主キー・外部キー）';
COMMENT ON COLUMN tag_view_history.viewed_at IS '閲覧日（複合主キー・日別集計）';

-- search_history: (query, searched_at) -> PRIMARY KEY
ALTER TABLE search_history DROP CONSTRAINT search_history_query_searched_at_key;
ALTER TABLE search_history DROP COLUMN id;
ALTER TABLE search_history ADD PRIMARY KEY (query, searched_at);
COMMENT ON COLUMN search_history.query IS '検索クエリ（複合主キー）';
COMMENT ON COLUMN search_history.searched_at IS '検索日（複合主キー・日別集計）';
