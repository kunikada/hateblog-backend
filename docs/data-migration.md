# 旧システムデータ移行（概要）

## 目的
旧システム（MySQL 8.0）から現行DB（PostgreSQL）へ、コアデータの移行方針を整理する。

## 前提
- 旧DB: MySQL 8.0
- 対象テーブル: `bookmarks`, `keywords`, `keyphrases`
- 現行スキーマ参照: `docs/database-schema.md`

## 対応関係（概要）
- `bookmarks` -> `entries`
- `keywords` -> `tags`
- `keyphrases` -> `entry_tags`

## 主要フィールドの対応（要確認）
- `bookmarks.link` + `bookmarks.sslp` -> `entries.url`（`sslp=1` は `https`、`sslp=0` は `http`。`link` はドメイン以降）
- `bookmarks.title` -> `entries.title`
- `bookmarks.description` -> `entries.excerpt`
- `bookmarks.subject` -> `entries.subject`
- `bookmarks.cnt` -> `entries.bookmark_count`
- `bookmarks.ientried` / `bookmarks.icreated` / `bookmarks.imodified` / `bookmarks.cdate` -> `entries.posted_at` / `entries.created_at` / `entries.updated_at`（`i*` は UnixTime の整数値のため変換。`cdate` は移行対象外）
- `keywords.keyword` -> `tags.name`
- `keyphrases.bookmark_id` -> `entry_tags.entry_id`
- `keyphrases.keyword_id` -> `entry_tags.tag_id`
- `keyphrases.score` -> `entry_tags.score`

## 変換ルール（概要）
- `entries.url`: `sslp` に応じて `http/https` を付与し、`link` はそのまま連結（正規化なし）
- `entries.posted_at`: `bookmarks.ientried` をUnixTimeから変換
- `entries.created_at`: `bookmarks.icreated` をUnixTimeから変換
- `entries.updated_at`: `bookmarks.imodified` をUnixTimeから変換

## 移行手順（概要）
- 抽出: 旧DBから `bookmarks` / `keywords` / `keyphrases` を取得（下記SQL）
- 変換: URL組み立てとUnixTime変換を適用（スクリプトで実施）
- 投入: `entries` / `tags` / `entry_tags` の順で登録（スクリプトで実施）
- 検証: 件数・重複・タグ紐づけを確認（スクリプトで実施）

## スクリプト実行
```sh
make migrator-run
```

または、バイナリを直接実行：
```sh
./bin/migrator
```

## 環境変数
`.env` ファイルで MySQL と PostgreSQL の接続情報を指定します：

### MySQL 設定
```
MYSQL_HOST=localhost
MYSQL_PORT=3306
MYSQL_USER=root
MYSQL_PASSWORD=
MYSQL_DB=hateblog_old
MYSQL_CONNECT_TIMEOUT=10s
```

### PostgreSQL 設定
```
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=hateblog
POSTGRES_PASSWORD=changeme
POSTGRES_DB=hateblog
POSTGRES_CONNECT_TIMEOUT=10s
```

## 処理の特徴
- **高速化**: Go による単一バイナリで実装（シェルスクリプト版は UUID 生成がボトルネック）
- **再開可能**: 移行先テーブルの行数で進捗を判定（途中中断時は続きから処理）
- **バッチ処理**: 1000行ごとにコミット（メモリとパフォーマンスのバランス）
- **進捗表示**: 各テーブルの処理状況を表示
  ```
  Total: 100000 | Already migrated: 50000 | Remaining: 50000
  [bookmarks] 51000/100000 (51.0%)
  ```


