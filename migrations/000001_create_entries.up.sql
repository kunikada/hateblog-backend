-- Create entries table
CREATE TABLE IF NOT EXISTS entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    url TEXT NOT NULL,
    posted_at TIMESTAMP WITH TIME ZONE NOT NULL,
    bookmark_count INTEGER NOT NULL DEFAULT 0,
    excerpt TEXT,
    subject TEXT,
    search_text TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- Constraints
    CONSTRAINT entries_url_unique UNIQUE (url),
    CONSTRAINT entries_bookmark_count_check CHECK (bookmark_count >= 0)
);

-- Create indexes
CREATE INDEX idx_entries_posted_at ON entries (posted_at DESC);
CREATE INDEX idx_entries_bookmark_count ON entries (bookmark_count DESC, posted_at DESC);
CREATE INDEX idx_entries_created_at ON entries (created_at);
CREATE INDEX idx_entries_updated_at ON entries (updated_at);

-- Add comment
COMMENT ON TABLE entries IS 'はてなブックマークのエントリー情報を格納するメインテーブル';
COMMENT ON COLUMN entries.id IS 'エントリーID（主キー）';
COMMENT ON COLUMN entries.title IS 'エントリーのタイトル';
COMMENT ON COLUMN entries.url IS 'エントリーのURL（ユニーク制約）';
COMMENT ON COLUMN entries.posted_at IS '記事の投稿日時';
COMMENT ON COLUMN entries.bookmark_count IS 'はてなブックマーク件数';
COMMENT ON COLUMN entries.excerpt IS '記事本文の抜粋';
COMMENT ON COLUMN entries.subject IS 'RSSフィードのsubject（画面非表示、内部利用）';
COMMENT ON COLUMN entries.search_text IS '検索用に結合したテキスト（title/excerpt/url、アプリ側で更新）';
COMMENT ON COLUMN entries.created_at IS 'レコード作成日時';
COMMENT ON COLUMN entries.updated_at IS 'レコード更新日時';
