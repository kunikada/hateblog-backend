# Database Schema

## 概要

hateblog バックエンドのデータベーススキーマ定義。PostgreSQL 18+ を使用。

## テーブル一覧

### コアデータ
- `entries` - はてなブックマークエントリー
- `tags` - タグマスター
- `entry_tags` - エントリーとタグの中間テーブル（スコア付き）

### 集計データ
- `click_metrics` - エントリークリック数の日別集計
- `tag_view_history` - タグ閲覧数の日別集計
- `search_history` - 検索キーワードの日別集計

### 認証/管理データ
- `api_keys` - API認証キー管理

---

## テーブル定義

### entries

はてなブックマークのエントリー情報を格納するメインテーブル。

| カラム名 | データ型 | NULL | デフォルト | 説明 |
|---------|---------|------|-----------|------|
| id | UUID | NOT NULL | gen_random_uuid() | エントリーID（主キー） |
| title | TEXT | NOT NULL | - | エントリーのタイトル |
| url | TEXT | NOT NULL | - | エントリーのURL（ユニーク制約） |
| posted_at | TIMESTAMP WITH TIME ZONE | NOT NULL | - | 記事の投稿日時 |
| bookmark_count | INTEGER | NOT NULL | 0 | はてなブックマーク件数 |
| excerpt | TEXT | NULL | - | 記事本文の抜粋 |
| subject | TEXT | NULL | - | RSSフィードのsubject（画面非表示、内部利用） |
| created_at | TIMESTAMP WITH TIME ZONE | NOT NULL | CURRENT_TIMESTAMP | レコード作成日時 |
| updated_at | TIMESTAMP WITH TIME ZONE | NOT NULL | CURRENT_TIMESTAMP | レコード更新日時 |

**制約:**
- PRIMARY KEY: `id`
- UNIQUE: `url`
- CHECK: `bookmark_count >= 0`

**インデックス:**
- `idx_entries_posted_at` - posted_at DESC（新着順リスト用）
- `idx_entries_bookmark_count` - bookmark_count DESC, posted_at DESC（人気順リスト用）
- `idx_entries_url` - url（ユニーク制約により自動作成）
- `idx_entries_created_at` - created_at（データ投入監視用）

**全文検索用インデックス（pg_bigm使用時）:**
- `idx_entries_title_gin` - GIN(title gin_bigm_ops)
- `idx_entries_excerpt_gin` - GIN(excerpt gin_bigm_ops)

**備考:**
- Faviconは、Google Favicon API (`https://www.google.com/s2/favicons?domain={domain}`) を使用して動的に取得するため、テーブルには格納しない
- `subject` はRSSフィード由来のメタデータで、画面表示には使用しないが、将来的な分析用に保持

---

### tags

タグのマスターテーブル。Yahoo! キーフレーズ抽出APIから取得したタグを格納。

| カラム名 | データ型 | NULL | デフォルト | 説明 |
|---------|---------|------|-----------|------|
| id | UUID | NOT NULL | gen_random_uuid() | タグID（主キー） |
| name | VARCHAR(100) | NOT NULL | - | タグ名（ユニーク制約） |
| created_at | TIMESTAMP WITH TIME ZONE | NOT NULL | CURRENT_TIMESTAMP | レコード作成日時 |

**制約:**
- PRIMARY KEY: `id`
- UNIQUE: `name`

**インデックス:**
- `idx_tags_name` - name（ユニーク制約により自動作成）

---

### entry_tags

エントリーとタグの多対多リレーションを管理する中間テーブル。スコア付き。

| カラム名 | データ型 | NULL | デフォルト | 説明 |
|---------|---------|------|-----------|------|
| entry_id | UUID | NOT NULL | - | エントリーID（外部キー） |
| tag_id | UUID | NOT NULL | - | タグID（外部キー） |
| score | REAL | NOT NULL | 0.0 | タグのスコア（Yahoo! APIから取得した重要度） |
| created_at | TIMESTAMP WITH TIME ZONE | NOT NULL | CURRENT_TIMESTAMP | レコード作成日時 |

**制約:**
- PRIMARY KEY: `(entry_id, tag_id)`
- FOREIGN KEY: `entry_id` REFERENCES `entries(id)` ON DELETE CASCADE
- FOREIGN KEY: `tag_id` REFERENCES `tags(id)` ON DELETE CASCADE
- CHECK: `score >= 0.0 AND score <= 1.0`

**インデックス:**
- `idx_entry_tags_entry_id` - entry_id（主キーにより自動作成）
- `idx_entry_tags_tag_id` - tag_id（タグ別エントリー一覧用）
- `idx_entry_tags_score` - entry_id, score DESC（エントリー内でのタグスコア順）

**備考:**
- `score` はYahoo! キーフレーズ抽出APIが返すスコア値（0.0〜1.0）を保存
- スコアが高いほど、そのエントリーにとって重要なタグであることを示す

---

### click_metrics

エントリーへのクリック数の日別集計テーブル。

| カラム名 | データ型 | NULL | デフォルト | 説明 |
|---------|---------|------|-----------|------|
| id | UUID | NOT NULL | gen_random_uuid() | 集計ID（主キー） |
| entry_id | UUID | NOT NULL | - | エントリーID（外部キー） |
| clicked_at | DATE | NOT NULL | - | クリック日（日別集計） |
| count | INTEGER | NOT NULL | 0 | クリック回数 |

**制約:**
- PRIMARY KEY: `id`
- UNIQUE: `(entry_id, clicked_at)`
- FOREIGN KEY: `entry_id` REFERENCES `entries(id)` ON DELETE CASCADE
- CHECK: `count >= 0`

**インデックス:**
- `idx_click_metrics_entry_date` - (entry_id, clicked_at)（ユニーク制約により自動作成）
- `idx_click_metrics_clicked_at` - clicked_at DESC（日付別集計用）

**備考:**
- 日別の集計値を保存（リアルタイムの個別クリックは記録しない）
- `ON CONFLICT (entry_id, clicked_at) DO UPDATE SET count = count + 1` で集計可能

---

### tag_view_history

タグ別エントリー一覧ページの閲覧数の日別集計テーブル。

| カラム名 | データ型 | NULL | デフォルト | 説明 |
|---------|---------|------|-----------|------|
| id | UUID | NOT NULL | gen_random_uuid() | 集計ID（主キー） |
| tag_id | UUID | NOT NULL | - | タグID（外部キー） |
| viewed_at | DATE | NOT NULL | - | 閲覧日（日別集計） |
| count | INTEGER | NOT NULL | 0 | 閲覧回数 |

**制約:**
- PRIMARY KEY: `id`
- UNIQUE: `(tag_id, viewed_at)`
- FOREIGN KEY: `tag_id` REFERENCES `tags(id)` ON DELETE CASCADE
- CHECK: `count >= 0`

**インデックス:**
- `idx_tag_view_history_tag_date` - (tag_id, viewed_at)（ユニーク制約により自動作成）
- `idx_tag_view_history_viewed_at` - viewed_at DESC（日付別集計用）

**備考:**
- タグ別エントリー一覧ページ（`/tags/{tag}/entries`）へのアクセスを日別で集計
- 人気タグの分析に使用

---

### search_history

検索キーワードの日別集計テーブル。

| カラム名 | データ型 | NULL | デフォルト | 説明 |
|---------|---------|------|-----------|------|
| id | UUID | NOT NULL | gen_random_uuid() | 集計ID（主キー） |
| query | TEXT | NOT NULL | - | 検索クエリ |
| searched_at | DATE | NOT NULL | - | 検索日（日別集計） |
| count | INTEGER | NOT NULL | 0 | 検索回数 |

**制約:**
- PRIMARY KEY: `id`
- UNIQUE: `(query, searched_at)`
- CHECK: `count >= 0`

**インデックス:**
- `idx_search_history_query_date` - (query, searched_at)（ユニーク制約により自動作成）
- `idx_search_history_searched_at` - searched_at DESC（日付別集計用）
- `idx_search_history_count` - count DESC（人気検索キーワード分析用）

**備考:**
- 全文検索（`GET /search?q={keyword}`）の実行回数を日別で集計
- 人気検索キーワードの分析やサジェスト機能に利用可能
- `ON CONFLICT (query, searched_at) DO UPDATE SET count = count + 1` で集計可能

---

### api_keys

API認証キーの管理テーブル。キーのハッシュ値と作成時のメタデータを保存。

| カラム名 | データ型 | NULL | デフォルト | 説明 |
|---------|---------|------|-----------|------|
| id | UUID | NOT NULL | gen_random_uuid() | APIキーID（主キー） |
| key_hash | TEXT | NOT NULL | - | APIキーのハッシュ値（bcryptまたはargon2） |
| name | VARCHAR(100) | NULL | - | キーの名前（管理用） |
| description | TEXT | NULL | - | キーの説明 |
| created_at | TIMESTAMP WITH TIME ZONE | NOT NULL | CURRENT_TIMESTAMP | レコード作成日時 |
| expires_at | TIMESTAMP WITH TIME ZONE | NULL | - | 有効期限（NULLの場合は無期限） |
| last_used_at | TIMESTAMP WITH TIME ZONE | NULL | - | 最終使用日時 |
| is_active | BOOLEAN | NOT NULL | true | 有効/無効フラグ |
| created_ip | INET | NULL | - | 作成時のIPアドレス |
| created_user_agent | TEXT | NULL | - | 作成時のUser-Agent |
| created_referrer | TEXT | NULL | - | 作成時のReferrer |

**制約:**
- PRIMARY KEY: `id`
- UNIQUE: `key_hash`
- CHECK: `expires_at IS NULL OR expires_at > created_at`

**インデックス:**
- `idx_api_keys_key_hash` - key_hash（ユニーク制約により自動作成）
- `idx_api_keys_is_active` - is_active（有効なキーのフィルタリング用）
- `idx_api_keys_expires_at` - expires_at（期限切れキーのクリーンアップ用）
- `idx_api_keys_last_used_at` - last_used_at DESC（使用状況分析用）

**備考:**
- 生のAPIキーはDBに保存せず、ハッシュ値のみを保存（セキュリティ強化）
- `key_hash` はbcryptまたはargon2でハッシュ化（比較的遅いアルゴリズムでブルートフォース対策）
- 発行時に一度だけクライアントに生のキーを返し、以降は照合時にハッシュで検証
- `is_active = false` で無効化（論理削除）
- `expires_at` でキーの有効期限を設定可能（将来の拡張用）
- `last_used_at` はAPI呼び出し時に非同期更新（パフォーマンス考慮）
- 作成時のメタデータ（IP、UA、Referrer）は不正利用の追跡や分析に使用

**API キーフォーマット:**
- プレフィックス付き: `hb_live_` + ランダム文字列（32文字以上）
- 例: `hb_live_1234567890abcdef1234567890abcdef`
- プレフィックスで環境を区別可能（`hb_test_` / `hb_live_`）

---

## ER図（テキスト表現）

```
┌──────────────────────┐
│       entries        │
├──────────────────────┤
│ id (PK, UUID)        │
│ title                │
│ url (UNIQUE)         │
│ posted_at            │
│ bookmark_count       │
│ excerpt              │
│ subject              │
│ created_at           │
│ updated_at           │
└──────┬───────────────┘
       │
       │ 1
       │
       │ N
┌──────┴───────────────┐
│     entry_tags       │
├──────────────────────┤
│ entry_id (PK,FK,UUID)│
│ tag_id (PK,FK,UUID)  │
│ score                │
│ created_at           │
└──────┬───────────────┘
       │
       │ N
       │
       │ 1
┌──────┴───────────────┐
│        tags          │
├──────────────────────┤
│ id (PK, UUID)        │
│ name (UNIQUE)        │
│ created_at           │
└──────────────────────┘

┌──────────────────────┐
│   click_metrics      │
├──────────────────────┤
│ id (PK, UUID)        │
│ entry_id (FK, UUID)  │────┐
│ clicked_at (DATE)    │    │
│ count                │    │ N:1
└──────────────────────┘    │
                            │
┌──────────────────────┐    │
│  tag_view_history    │    │
├──────────────────────┤    │
│ id (PK, UUID)        │    │
│ tag_id (FK, UUID)    │──┐ │
│ viewed_at (DATE)     │  │ │
│ count                │  │ │
└──────────────────────┘  │ │
                          │ │
                          │ │
                    ┌─────┴─┴─────┐
                    │   entries   │
                    │     tags    │
                    └─────────────┘

┌──────────────────────┐
│   search_history     │
├──────────────────────┤
│ id (PK, UUID)        │
│ query                │
│ searched_at (DATE)   │
│ count                │
└──────────────────────┘

┌──────────────────────┐
│      api_keys        │
├──────────────────────┤
│ id (PK, UUID)        │
│ key_hash (UNIQUE)    │
│ name                 │
│ description          │
│ created_at           │
│ expires_at           │
│ last_used_at         │
│ is_active            │
│ created_ip           │
│ created_user_agent   │
│ created_referrer     │
└──────────────────────┘
```

---

## データ型の選択理由

### UUID vs SERIAL
- 全てのテーブルの主キーに UUID を使用
- **理由**:
  - グローバルに一意な識別子（分散システムでの衝突回避）
  - 予測不可能性（セキュリティ向上、IDから総レコード数を推測されない）
  - マイクロサービス化やデータ移行時の柔軟性
  - 外部APIとの連携時にIDの衝突を防ぐ
- **デメリット**:
  - 16バイト（128bit）でSERIALの8バイトより大きい
  - インデックスサイズが大きくなる
  - 人間が読みにくい
- **採用判断**: 将来的な拡張性とセキュリティを優先し、UUID を採用

### gen_random_uuid() vs uuid_generate_v4()
- PostgreSQL 13+ の組み込み関数 `gen_random_uuid()` を使用
- `uuid-ossp` 拡張不要でシンプル

### TEXT vs VARCHAR
- `entries.title`, `entries.url`, `entries.excerpt`, `entries.subject` は TEXT を使用
- 長さ制限が不明確なものは TEXT で柔軟に対応
- `tags.name` は VARCHAR(100) で制限（タグ名は通常短い）

### TIMESTAMP WITH TIME ZONE
- すべての日時カラムで使用
- タイムゾーンを保持することで国際化に対応
- `posted_at` - 記事の投稿日時（はてなブックマークから取得）
- `created_at`, `updated_at` - レコードのライフサイクル管理
- `viewed_at`, `searched_at` - ユーザー行動の記録

### INTEGER vs BIGINT
- `entries.bookmark_count` は INTEGER
- ブックマーク数は現実的に INT4 の範囲内（最大21億）
- `search_history.result_count` も INTEGER（検索結果件数）

---

## インデックス戦略

### 新着順リスト（`GET /entries/new`）
```sql
-- クエリ例
SELECT * FROM entries
WHERE bookmark_count >= ?
ORDER BY posted_at DESC
LIMIT ? OFFSET ?;

-- 使用インデックス
idx_entries_posted_at
```

### 人気順リスト（`GET /entries/hot`）
```sql
-- クエリ例
SELECT * FROM entries
WHERE bookmark_count >= ?
ORDER BY bookmark_count DESC, posted_at DESC
LIMIT ? OFFSET ?;

-- 使用インデックス
idx_entries_bookmark_count (複合インデックス)
```

### タグ別エントリー一覧（`GET /tags/{tag}/entries`）
```sql
-- クエリ例
SELECT e.* FROM entries e
INNER JOIN entry_tags et ON e.id = et.entry_id
INNER JOIN tags t ON et.tag_id = t.id
WHERE t.name = ?
ORDER BY e.posted_at DESC;

-- 使用インデックス
idx_entry_tags_tag_id, idx_entries_posted_at
```

### URL重複チェック（データ投入時）
```sql
-- クエリ例
INSERT INTO entries (url, ...) VALUES (?, ...)
ON CONFLICT (url) DO UPDATE SET bookmark_count = EXCLUDED.bookmark_count;

-- 使用インデックス
entries_url_key (UNIQUE制約)
```

### 全文検索（`GET /search?q={keyword}`）
```sql
-- pg_bigm使用時のクエリ例
SELECT * FROM entries
WHERE title LIKE '%' || ? || '%'
   OR excerpt LIKE '%' || ? || '%'
ORDER BY bookmark_count DESC;

-- 使用インデックス
idx_entries_title_gin, idx_entries_excerpt_gin
```

---

## マイグレーション方針

### 1. 初期マイグレーション
- `000001_create_entries.up.sql` - entries テーブル作成
- `000002_create_tags.up.sql` - tags テーブル作成
- `000003_create_entry_tags.up.sql` - entry_tags テーブル作成（score付き）
- `000004_create_click_metrics.up.sql` - click_metrics テーブル作成（日別集計）
- `000005_create_tag_view_history.up.sql` - tag_view_history テーブル作成（日別集計）
- `000006_create_search_history.up.sql` - search_history テーブル作成（日別集計）
- `000007_create_api_keys.up.sql` - api_keys テーブル作成（API認証管理）

### 2. 全文検索マイグレーション（pg_bigm導入時）
- `000008_enable_pg_bigm.up.sql` - pg_bigm 拡張の有効化
- `000009_create_fulltext_indexes.up.sql` - GINインデックスの作成

### 3. ロールバック（down）
- 各マイグレーションに対応する `.down.sql` を用意
- CASCADE で依存関係を考慮した削除

---

## パフォーマンスチューニング

### 想定データ量
- entries: 100万レコード/年（継続的に増加）
- tags: 1万レコード（緩やかに増加）
- entry_tags: 300万レコード/年（entries の 3倍想定、タグ×エントリー）
- click_metrics: 365万レコード/年（エントリー100万 × 365日の最大値、実際は稼働日のみ）
- tag_view_history: 36万レコード/年（1万タグ × 365日の最大値、実際はアクセスされたもののみ）
- search_history: 数万〜数十万レコード/年（ユニークな検索クエリ × 日数）
- api_keys: 数十〜数百レコード（発行されるAPIキーの総数、緩やかに増加）

### パーティショニング検討
- `click_metrics` は月次パーティショニングを検討（データ量が大きい場合）
- `tag_view_history` は年次パーティショニングで十分
- `search_history` は年次パーティショニングで十分
- `entries` は当面パーティショニング不要（posted_at でアーカイブは可能）

### データ保持期間
- `click_metrics`: 2年間保持（統計分析用）
- `tag_view_history`: 1年間保持
- `search_history`: 1年間保持
- `api_keys`: 永続保持（`is_active = false` で論理削除、定期的に物理削除可能）
- コアデータ（entries, tags, entry_tags）: 永続保持

### VACUUM/ANALYZE
- 定期的な VACUUM ANALYZE の実行（cron or pg_cron）
- 集計テーブルは UPDATE が多いため、autovacuum の閾値を調整
- データ削除後に VACUUM を実行

### 統計情報
- `bookmark_count` の分布に偏りがあるため、統計情報の精度を上げる
```sql
ALTER TABLE entries ALTER COLUMN bookmark_count SET STATISTICS 1000;
```

---

## バックアップ戦略

### 物理バックアップ
- `pgBackRest` または `wal-g` を使用
- 日次フルバックアップ + 継続的 WAL アーカイブ
- ポイントインタイムリカバリ（PITR）対応

### 論理バックアップ
- 週次で `pg_dump` による論理バックアップ
- 開発環境へのリストア用

### Redis
- データは PostgreSQL から再構築可能
- RDB/AOF は無効化してパフォーマンス優先

---

## セキュリティ

### アクセス制御
- アプリケーション用ユーザー（`hateblog_app`）
  - entries: SELECT, INSERT, UPDATE
  - tags: SELECT, INSERT
  - entry_tags: SELECT, INSERT, DELETE
  - click_metrics: SELECT, INSERT, UPDATE（集計値の更新）
  - tag_view_history: SELECT, INSERT, UPDATE（集計値の更新）
  - search_history: SELECT, INSERT, UPDATE（集計値の更新）
  - api_keys: SELECT, INSERT, UPDATE（キー発行・検証・最終使用日時更新）
- 管理用ユーザー（`hateblog_admin`）
  - 全テーブル: ALL PRIVILEGES

### 個人情報・プライバシー
- 個人を特定できる情報（ユーザーID、IPアドレス、Cookie）は一切保存しない
- 集計データのみを保存（GDPR/CCPA対応）
- **例外**: `api_keys` テーブルのみ、APIキー発行時のIPアドレス・User-Agent・Referrerを記録（不正利用の追跡・セキュリティ監査のため）

### SQLインジェクション対策
- プレースホルダーを必ず使用（`pgx` のパラメータバインド）
- 動的SQLは避ける

### 監査ログ
- PostgreSQL の `pg_audit` 拡張を検討（本番環境）
- DDL/DML の実行履歴を記録

---

## 関連ドキュメント

- [IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md) - 実装計画
- [ARCHITECTURE.md](ARCHITECTURE.md) - アーキテクチャ設計
- [FULLTEXT_SEARCH_COMPARISON.md](FULLTEXT_SEARCH_COMPARISON.md) - 全文検索実装の比較
