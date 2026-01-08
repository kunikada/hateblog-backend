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
chmod +x scripts/migrate_tsv.sh
./scripts/migrate_tsv.sh
```

## 投入（psql）
接続に必要なオプションは適宜指定する。
```sh
psql -v ON_ERROR_STOP=1 <DB_NAME>
```

```sql
\copy entries(id,title,url,posted_at,bookmark_count,excerpt,subject,created_at,updated_at) FROM 'entries.tsv' WITH (FORMAT csv, DELIMITER E'\t', HEADER true, NULL '\N')
\copy tags(id,name,created_at) FROM 'tags.tsv' WITH (FORMAT csv, DELIMITER E'\t', HEADER true, NULL '\N')
\copy entry_tags(entry_id,tag_id,score,created_at) FROM 'entry_tags.tsv' WITH (FORMAT csv, DELIMITER E'\t', HEADER true, NULL '\N')
```

## 検証（概要）
- `bookmarks` と `entries` の行数が一致すること
- `keywords` と `tags` の行数が一致すること
- `keyphrases` と `entry_tags` の行数が一致すること

## 検証コマンド
```sh
mysql --batch --raw --default-character-set=utf8mb4 -e "
SELECT COUNT(*) FROM bookmarks;
SELECT COUNT(*) FROM keywords;
SELECT COUNT(*) FROM keyphrases;
" hateblog
```

```sh
psql -v ON_ERROR_STOP=1 <DB_NAME>
```

```sql
SELECT COUNT(*) FROM entries;
SELECT COUNT(*) FROM tags;
SELECT COUNT(*) FROM entry_tags;
```

## 抽出（MySQLコマンド）
接続に必要なオプションは適宜指定する。
```sh
mysql --batch --raw --default-character-set=utf8mb4 -e "
SELECT
  id,
  title,
  link,
  sslp,
  description,
  subject,
  cnt,
  ientried,
  icreated,
  imodified
FROM bookmarks;
" hateblog > bookmarks.tsv
```

```sh
mysql --batch --raw --default-character-set=utf8mb4 -e "
SELECT
  id,
  keyword,
  bookmark_cnt
FROM keywords;
" hateblog > keywords.tsv
```

```sh
mysql --batch --raw --default-character-set=utf8mb4 -e "
SELECT
  bookmark_id,
  keyword_id,
  score
FROM keyphrases;
" hateblog > keyphrases.tsv
```

## スクリプト入出力（概要）
- 入力: 抽出したTSV（UTF-8、ヘッダあり）
- 出力: `entries` / `tags` / `entry_tags` への投入用データ
