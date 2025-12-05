# Implementation Plan

## 概要

hateblog バックエンドAPIの段階的な実装計画。

## 前提

- Go 1.25、PostgreSQL 18、Redis 8.4 のDocker環境が構築済み
- OpenAPI 3.1 で API 仕様を定義し、oapi-codegen でコード生成
- データソースははてなブックマークAPIまたはクローリング（実装段階で確定）

## フェーズ1: 基盤構築

### 1.1 プラットフォーム層の実装

**目的**: 設定読み込み、ログ、DB接続、Redis接続の基盤整備

**実装内容**:
- [x] `internal/platform/config` - 環境変数からの設定読み込み（`caarlos0/env/v10`）
- [x] `internal/platform/logger` - 構造化ログ設定（`log/slog` + `tint`）
- [x] `internal/platform/database` - PostgreSQL接続プール（`pgx/v5`）
- [x] `internal/platform/cache` - Redis接続（`go-redis/v9`）
- [x] `internal/platform/server` - HTTPサーバー設定

**検証**:
- [x] 環境変数読み込みのテスト
- [x] DB接続確認
- [x] Redis接続確認

### 1.2 OpenAPI 仕様の定義

**目的**: API仕様の明確化とコード生成の準備

**実装内容**:
- [x] `openapi.yaml` - OpenAPI 3.0 仕様ファイル作成（全エンドポイント定義済み）
  - [x] `/health` エンドポイント
  - [x] `/entries/new` - 新着順リスト
  - [x] `/entries/hot` - 人気順リスト
  - [x] スキーマ定義（Entry、Pagination、Error）
  - [x] その他全エンドポイント（アーカイブ、ランキング、タグ、検索、計測、etc.）
- [x] `oapi-codegen/v2` でサーバースタブ生成設定
- [x] Makefile に生成タスク追加（`make generate`）
- [x] `oapi-codegen.yaml` 設定ファイル作成

**検証**:
- [ ] OpenAPI 仕様の妥当性検証（Swagger Editor）
- [ ] 生成コードのコンパイル確認（次フェーズで実施）

### 1.3 DBスキーマ設計とマイグレーション

**目的**: データモデルの確立

**実装内容**:
- [x] `migrations/` 配下にマイグレーションファイル作成
  - [x] `entries` テーブル（id, title, url, posted_at, bookmark_count, excerpt, subject, created_at, updated_at）
  - [x] `tags` テーブル（id, name）
  - [x] `entry_tags` テーブル（entry_id, tag_id, score）
  - [x] `click_metrics` テーブル（id, entry_id, clicked_at, count）
  - [x] `tag_view_history` テーブル（id, tag_id, viewed_at, count）
  - [x] `search_history` テーブル（id, query, searched_at, count）
  - [x] `api_keys` テーブル（id, key_hash, name, description, etc.）
  - [x] インデックス設定（posted_at, bookmark_count, url unique）
  - [x] `pg_bigm` 拡張機能とGINインデックス
- [x] `golang-migrate` 対応のup/downファイル

**検証**:
- [ ] マイグレーション適用確認（次フェーズで実施）
- [ ] ロールバック確認（次フェーズで実施）

## フェーズ2: コア機能実装

### 2.1 ドメイン層の実装

**目的**: ビジネスロジックの中核を定義

**実装内容**:
- [ ] `internal/domain/entry` - Entry エンティティ、ValueObject
- [ ] `internal/domain/tag` - Tag エンティティ
- [ ] `internal/domain/repository` - Repository インターフェース定義
  - [ ] `EntryRepository` - CRUD + フィルタ/ソートメソッド
  - [ ] `TagRepository` - CRUD + 検索メソッド

**検証**:
- [ ] ドメインモデルの単体テスト

### 2.2 インフラ層の実装

**目的**: DB/外部APIとの接続実装

**実装内容**:
- [ ] `internal/infra/postgres` - PostgreSQL リポジトリ実装
  - [ ] EntryRepository 実装
  - [ ] TagRepository 実装
- [ ] `internal/infra/redis` - Redis キャッシュ実装
  - [ ] エントリーリストのキャッシュ
  - [ ] TTL 設定（1時間など）
- [ ] `internal/infra/hatena` - はてなブックマークAPI クライアント（将来実装）

**検証**:
- [ ] リポジトリの統合テスト（testcontainers）
- [ ] キャッシュの動作確認

### 2.3 アプリケーション層の実装

**目的**: ユースケースの実装

**実装内容**:
- [ ] `internal/app/entry` - エントリー関連ユースケース
  - [ ] `ListNewEntries` - 新着順リスト取得（キャッシュ付き）
  - [ ] `ListHotEntries` - 人気順リスト取得（キャッシュ付き）
  - [ ] フィルタリング（最小ブックマーク件数）
  - [ ] ページネーション（offset/limit）

**検証**:
- [ ] ユースケースの単体テスト（モックリポジトリ）
- [ ] キャッシュヒット/ミスの確認

### 2.4 ハンドラ層の実装

**目的**: HTTPエンドポイントの実装

**実装内容**:
- [ ] `internal/infra/handler` - OpenAPI生成インターフェースの実装
  - [ ] `NewEntriesHandler` - GET /entries/new
  - [ ] `HotEntriesHandler` - GET /entries/hot
  - [ ] `HealthCheckHandler` - GET /health
- [ ] エラーハンドリング（統一エラーレスポンス）
- [ ] バリデーション（クエリパラメータ）

**検証**:
- [ ] HTTPテスト（`httptest` パッケージ）
- [ ] E2Eテスト（curl/HTTPクライアント）

## フェーズ3: 追加機能実装

### 3.1 アーカイブ機能

**実装内容**:
- [ ] API エンドポイント: `GET /archive`
- [ ] 年月日での階層フィルタ
- [ ] 日付ごとのエントリー件数集計

**検証**:
- [ ] アーカイブAPIの動作確認

### 3.2 ランキング機能

**実装内容**:
- [ ] API エンドポイント:
  - [ ] `GET /rankings/yearly`
  - [ ] `GET /rankings/monthly`
  - [ ] `GET /rankings/weekly`
- [ ] ブックマーク件数でのソート
- [ ] 期間指定（年、月、週）

**検証**:
- [ ] ランキングAPIの動作確認

### 3.3 タグ機能

**実装内容**:
- [ ] API エンドポイント:
  - [ ] `GET /tags/{tag}/entries` - タグ別エントリー一覧
  - [ ] `GET /tags` - タグ一覧取得
- [ ] タグ別エントリー一覧取得
- [ ] タグの正規化（小文字統一など）
- [ ] タグ閲覧履歴の記録

**検証**:
- [ ] タグAPIの動作確認
- [ ] タグ閲覧履歴の記録確認

### 3.4 検索機能

**実装内容**:
- [ ] API エンドポイント: `GET /search?q={keyword}`
- [ ] 全文検索（タイトル、抜粋、タグ）
- [ ] 検索履歴の記録（日別集計）
- **実装方式**: PostgreSQL + `pg_bigm` 拡張
  - [x] `pg_bigm` 拡張のインストール（マイグレーション完了）
  - [x] GINインデックスの作成（title, excerpt - マイグレーション完了）
  - [ ] 検索APIの実装（LIKE + ブックマーク件数ソート）
  - 将来的にElasticsearchへの移行も可能

**検証**:
- [ ] 全文検索の動作確認
- [ ] 日本語検索の精度確認
- [ ] 検索履歴の記録確認

**詳細**: [FULLTEXT_SEARCH_COMPARISON.md](FULLTEXT_SEARCH_COMPARISON.md) 参照

### 3.5 クリック計測

**実装内容**:
- [ ] API エンドポイント: `POST /metrics/clicks`
- [ ] エントリーIDとタイムスタンプの記録
- [ ] 日別集計処理（UPSERT with ON CONFLICT）
- [x] 集計用テーブル（`click_metrics` - マイグレーション完了）

**検証**:
- [ ] クリック計測APIの動作確認
- [ ] 日別集計の動作確認

### 3.6 Favicon プロキシ

**目的**: ドメインのfaviconを取得・キャッシュするプロキシAPI

**実装内容**:
- [ ] API エンドポイント: `GET /favicons?domain={domain}`
- [ ] Google の favicon サービスを利用: `https://www.google.com/s2/favicons?domain={domain}`
- [ ] Redisにキャッシュ（TTL: 7日間など長めに設定）
- [ ] アクセス制限:
  - [ ] 同一ドメインへのリクエストは1秒以内に1回まで（Redis + rate limiting）
  - [ ] キャッシュヒット時は外部APIを叩かない
- [ ] エラーハンドリング:
  - [ ] Google APIが落ちている場合のフォールバック（デフォルトアイコン）
  - [ ] タイムアウト設定（3秒）

**キャッシュキー設計**:
- `favicon:{domain}` - favicon画像データ（バイナリ）
- `favicon:ratelimit:{domain}` - レート制限用（TTL: 1秒）

**検証**:
- [ ] キャッシュヒット/ミスの確認
- [ ] レート制限の動作確認
- [ ] 外部API障害時のフォールバック確認

## フェーズ4: 最適化とセキュリティ

### 4.1 パフォーマンス最適化

**実装内容**:
- [ ] クエリのチューニング
- [ ] インデックスの最適化
- [ ] N+1問題の解消
- [ ] レスポンスサイズの削減
- [ ] Redis キャッシュ戦略の見直し

**検証**:
- [ ] 負荷テスト実施
- [ ] クエリパフォーマンス測定

### 4.2 セキュリティ強化

**実装内容**:
- [ ] Rate limiting（Redis ベース）
- [x] CORS 設定（middleware実装済み）
- [x] セキュリティヘッダー（middleware実装済み）
- [ ] SQLインジェクション対策の確認
- [ ] XSS対策（出力エスケープ）
- [x] `gosec` / `govulncheck` での脆弱性チェック（Makefile設定済み）

**検証**:
- [ ] セキュリティスキャン実施
- [ ] ペネトレーションテスト

### 4.3 監視・ログ

**実装内容**:
- [ ] Prometheus メトリクス（`promhttp`）
- [x] アクセスログ（構造化 - middleware実装済み）
- [ ] エラー率・レイテンシの監視
- [ ] アラート設定（将来）

**検証**:
- [ ] メトリクス収集の確認
- [ ] ログ出力の確認

## フェーズ5: 運用準備

### 5.1 CI/CD パイプライン

**実装内容**:
- [ ] GitHub Actions（または GitLab CI）設定
  - [x] リント（`golangci-lint` - Makefile設定済み）
  - [x] テスト（`go test -race` - Makefile設定済み）
  - [x] セキュリティスキャン（`gosec`, `govulncheck`, `trivy` - Makefile設定済み）
  - [ ] ビルド（Docker イメージ）
  - [ ] デプロイ（VPS への自動デプロイ）

**検証**:
- [ ] CI/CDパイプラインの動作確認

### 5.2 ドキュメント整備

**実装内容**:
- [x] API ドキュメント（OpenAPI仕様作成済み）
- [ ] 運用マニュアル
- [ ] トラブルシューティングガイド

**検証**:
- [ ] ドキュメントの完全性確認

### 5.3 データ投入

**目的**: はてなブックマークの公開フィードから定期的にエントリーを取得

**データソース**:
- `https://b.hatena.ne.jp/entrylist?sort=hot&mode=rss&threshold=5`
- `https://feeds.feedburner.com/hatena/b/hotentry`

**実装内容**:
- [ ] RSSフィードパーサーの実装（`github.com/mmcdole/gofeed` など）
- [ ] フィード取得スクリプト（`cmd/fetcher/main.go`）
  - [ ] 各フィードURLからエントリー取得
  - [ ] パース（タイトル、URL、ブックマーク件数、投稿日時）
  - [ ] DB格納（重複チェック: URL unique制約）
  - [ ] favicon URLの生成（ドメイン抽出）
  - [ ] **タグの抽出**: Yahoo! キーフレーズ抽出API
    - API: `https://developer.yahoo.co.jp/webapi/jlp/keyphrase/v2/extract.html`
    - エントリーのタイトルと抜粋からキーフレーズを抽出
    - 上位3-5個のキーフレーズをタグとして登録
    - レート制限対応（リクエスト間隔の制御）
    - [x] APIキーの環境変数管理（`YAHOO_API_KEY` - .env.example設定済み）
- [ ] **ブックマーク件数更新スクリプト（複数バッチの並行実行）**（`cmd/updater/main.go`）
  - はてなブックマーク一括取得API: `https://bookmark.hatenaapis.com/count/entries?url=url1&url=url2&...`
  - 最大50URL/リクエストで効率的に取得
  - [ ] 優先度別バッチ戦略（それぞれ独立したバッチとして実行）:
    - [ ] **高優先度バッチ**（15分ごと、50件×2回）
      - 条件: `WHERE (posted_at > NOW() - INTERVAL '30 days' OR bookmark_count >= 100) ORDER BY updated_at ASC LIMIT 50`
      - 対象: 新着記事とバズった記事を高頻度で更新
    - [ ] **中優先度バッチ**（1時間ごと、50件）
      - 条件: `WHERE (posted_at > NOW() - INTERVAL '90 days' OR bookmark_count >= 20) ORDER BY updated_at ASC LIMIT 50`
      - 対象: 中堅記事を中頻度で更新
    - [ ] **低優先度バッチ**（3時間ごと、50件）
      - 条件: `WHERE (posted_at > NOW() - INTERVAL '180 days' OR bookmark_count >= 10) ORDER BY updated_at ASC LIMIT 50`
      - 対象: 古めの記事を低頻度で更新
    - [ ] **全体循環バッチ**（6時間ごと、50件）
      - 条件: `WHERE bookmark_count >= 5 ORDER BY updated_at ASC LIMIT 50`
      - 対象: 最低限のフィルタで全体を循環更新
  - [ ] updated_at基準の循環更新により、全エントリーをまんべんなく更新
  - [ ] 合計: 約11,400件/日のペースで更新（100万件規模に対応）
  - [ ] 更新時にブックマーク件数とupdated_atを同時に更新
- [ ] 定期実行
  - [ ] cron（`*/15 * * * *` 15分ごと）または
  - [ ] systemd timer
  - [ ] または Goアプリ内スケジューラ（`github.com/robfig/cron/v3`）
- [ ] 差分更新ロジック
  - [ ] 既存エントリーのブックマーク件数更新
  - [ ] 新規エントリーのみINSERT
- [ ] エラーハンドリング
  - [ ] フィード取得失敗時のリトライ（指数バックオフ）
  - [ ] パースエラーのログ記録
  - [ ] DB接続エラー時の再試行
  - [ ] API障害時のスキップとログ記録

**検証**:
- [ ] 手動実行でフィード取得確認
- [ ] 重複エントリーの除外確認
- [ ] ブックマーク件数の更新確認
- [ ] 優先度別バッチの動作確認
  - [ ] updated_atが正しく更新されること
  - [ ] 各優先度の条件で正しくエントリーが取得されること
  - [ ] 一括API（50件/リクエスト）が正常に動作すること

## 実装順序の推奨

1. **最優先**: フェーズ1（基盤構築）
2. **高優先**: フェーズ2（コア機能）- 新着/人気順リストまで
3. **中優先**: フェーズ3（追加機能）- アーカイブ、ランキング、タグ、Faviconプロキシ
4. **低優先**: フェーズ3 - 検索、クリック計測
5. **並行実施**: フェーズ4（最適化）、フェーズ5（運用準備）

## 各フェーズの想定工数

- フェーズ1: 3-5日
- フェーズ2: 5-7日
- フェーズ3.1-3.3: 3-4日
- フェーズ3.4-3.5: 2-3日
- フェーズ4: 3-5日
- フェーズ5: 2-3日

**合計**: 約 3-4週間（1人での実装を想定）

## 確定事項

- [x] **favicon取得**: Google favicon サービス (`https://www.google.com/s2/favicons?domain=`) + Redisキャッシュプロキシ
- [x] **全文検索**: PostgreSQL + `pg_bigm` 拡張を採用（初期実装）。将来的にElasticsearchへの移行も検討可能。
- [x] **データ取得**: はてなブックマークの公開RSSフィードから取得
  - `https://b.hatena.ne.jp/entrylist?sort=hot&mode=rss&threshold=5`
  - `https://feeds.feedburner.com/hatena/b/hotentry`
  - 15分ごとに定期実行
- [x] **タグ抽出**: Yahoo! キーフレーズ抽出API (`https://developer.yahoo.co.jp/webapi/jlp/keyphrase/v2/extract.html`)
  - タイトルと抜粋から上位3-5個のキーフレーズを抽出
  - 環境変数 `YAHOO_API_KEY` でAPIキー管理

## 未確定事項

- [ ] フロントエンドとのAPI仕様調整

## 参考資料

- [ARCHITECTURE.md](ARCHITECTURE.md) - 技術アーキテクチャ
- [FULLTEXT_SEARCH_COMPARISON.md](FULLTEXT_SEARCH_COMPARISON.md) - 全文検索実装の比較検討
- [features.md](features.md) - 機能一覧
- [tech-requirements.md](tech-requirements.md) - 技術要件
- [CONTRIBUTING.md](../CONTRIBUTING.md) - コントリビューションガイド
