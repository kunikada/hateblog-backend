-- Create tags table
CREATE TABLE IF NOT EXISTS tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    CONSTRAINT tags_name_unique UNIQUE (name)
);

-- Add comment
COMMENT ON TABLE tags IS 'タグのマスターテーブル。Yahoo! キーフレーズ抽出APIから取得したタグを格納';
COMMENT ON COLUMN tags.id IS 'タグID（主キー）';
COMMENT ON COLUMN tags.name IS 'タグ名（ユニーク制約）';
COMMENT ON COLUMN tags.created_at IS 'レコード作成日時';
