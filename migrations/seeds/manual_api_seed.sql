-- 手動APIテスト用の最小シードデータ
-- 実行方法: scripts/seed_manual.sh を参照

BEGIN;

-- 依存関係を崩さないようTRUNCATE順に注意
TRUNCATE TABLE entry_tags, click_metrics, tag_view_history, search_history, tags, entries RESTART IDENTITY CASCADE;

-- タグ
INSERT INTO tags (id, name, created_at) VALUES
    ('11111111-1111-1111-1111-111111111111', 'go',         '2024-12-01T00:00:00Z'),
    ('22222222-2222-2222-2222-222222222222', 'ai',         '2024-12-01T00:00:00Z'),
    ('33333333-3333-3333-3333-333333333333', 'cloud',      '2024-12-01T00:00:00Z'),
    ('44444444-4444-4444-4444-444444444444', 'database',   '2024-12-01T00:00:00Z');

-- エントリー（2日分）
INSERT INTO entries (id, title, url, posted_at, bookmark_count, excerpt, subject, search_text, created_at, updated_at) VALUES
    ('aaaaaaa1-aaaa-aaaa-aaaa-aaaaaaaaaaa1',
     'Goで作る高速API入門',
     'https://example.com/articles/go-fast-api',
     '2024-12-02T02:30:00Z',
     120,
     'GoとPostgreSQLで作るシンプルなAPIの実装手順。',
     'tech',
     'Goで作る高速API入門 GoとPostgreSQLで作るシンプルなAPIの実装手順。 https://example.com/articles/go-fast-api',
     '2024-12-02T02:30:00Z',
     '2024-12-02T02:30:00Z'),
    ('aaaaaaa2-aaaa-aaaa-aaaa-aaaaaaaaaaa2',
     '大規模言語モデルの活用パターン',
     'https://example.com/articles/llm-patterns',
     '2024-12-02T05:10:00Z',
     45,
     'LLMを検索・要約・分類に使う際の実装例。',
     'ai',
     '大規模言語モデルの活用パターン LLMを検索・要約・分類に使う際の実装例。 https://example.com/articles/llm-patterns',
     '2024-12-02T05:10:00Z',
     '2024-12-02T05:10:00Z'),
    ('aaaaaaa3-aaaa-aaaa-aaaa-aaaaaaaaaaa3',
     'クラウドコスト最適化の基礎',
     'https://example.com/articles/cloud-cost',
     '2024-12-01T03:00:00Z',
     30,
     '小規模サービスでできるコスト最適化のチェックリスト。',
     'cloud',
     'クラウドコスト最適化の基礎 小規模サービスでできるコスト最適化のチェックリスト。 https://example.com/articles/cloud-cost',
     '2024-12-01T03:00:00Z',
     '2024-12-01T03:00:00Z'),
    ('aaaaaaa4-aaaa-aaaa-aaaa-aaaaaaaaaaa4',
     'PostgreSQLチューニングの第一歩',
     'https://example.com/articles/pg-tuning',
     '2024-12-01T07:45:00Z',
     12,
     'インデックス設計と基本的なVACUUM設定のポイント。',
     'database',
     'PostgreSQLチューニングの第一歩 インデックス設計と基本的なVACUUM設定のポイント。 https://example.com/articles/pg-tuning',
     '2024-12-01T07:45:00Z',
     '2024-12-01T07:45:00Z');

-- エントリーとタグの紐付け
INSERT INTO entry_tags (entry_id, tag_id, score, created_at) VALUES
    ('aaaaaaa1-aaaa-aaaa-aaaa-aaaaaaaaaaa1', '11111111-1111-1111-1111-111111111111', 90, '2024-12-02T02:30:00Z'),
    ('aaaaaaa1-aaaa-aaaa-aaaa-aaaaaaaaaaa1', '33333333-3333-3333-3333-333333333333', 60, '2024-12-02T02:30:00Z'),
    ('aaaaaaa2-aaaa-aaaa-aaaa-aaaaaaaaaaa2', '22222222-2222-2222-2222-222222222222', 85, '2024-12-02T05:10:00Z'),
    ('aaaaaaa2-aaaa-aaaa-aaaa-aaaaaaaaaaa2', '11111111-1111-1111-1111-111111111111', 50, '2024-12-02T05:10:00Z'),
    ('aaaaaaa3-aaaa-aaaa-aaaa-aaaaaaaaaaa3', '33333333-3333-3333-3333-333333333333', 80, '2024-12-01T03:00:00Z'),
    ('aaaaaaa4-aaaa-aaaa-aaaa-aaaaaaaaaaa4', '44444444-4444-4444-4444-444444444444', 85, '2024-12-01T07:45:00Z');

-- クリック計測（/metrics/clicks を試す場合の確認用）
INSERT INTO click_metrics (id, entry_id, clicked_at, count) VALUES
    ('bbbbbbb1-bbbb-bbbb-bbbb-bbbbbbbbbbb1', 'aaaaaaa1-aaaa-aaaa-aaaa-aaaaaaaaaaa1', '2024-12-02', 5),
    ('bbbbbbb2-bbbb-bbbb-bbbb-bbbbbbbbbbb2', 'aaaaaaa2-aaaa-aaaa-aaaa-aaaaaaaaaaa2', '2024-12-02', 2),
    ('bbbbbbb3-bbbb-bbbb-bbbb-bbbbbbbbbbb3', 'aaaaaaa3-aaaa-aaaa-aaaa-aaaaaaaaaaa3', '2024-12-01', 1);

COMMIT;
