# 作業ルール

手戻りによる工数の増加を防ぐため、以下のルールに従って作業を進める
応答は必ず日本語とする

## ドキュメント

- 必要最低限の内容にする
- 1つのファイルは200から400行程度、最大でも500行に収める
- サンプルや事例を記載する必要があると判断したら事前確認する
- 要点をまとめた文章とサンプルや事例のファイルは別にする
- 記述する対象のファイルが正しいか、重複していないか確認する

## 実装

- 指示と違うことを行う場合は事前確認して同意を得る
- 設定や方針を変更する場合は事前確認して同意を得る
- 作業中に別の問題が見つかった場合は中止して事前確認する
- 作業中に作成した一時ファイルは必ず消す
- デバッグメッセージを追加するときはデバッグレベルで出力する
- データやタイプではなく機能やドメインを考慮してディレクトリやファイルを構成する
- 実装後にformat/lint/testなどのコマンドは実行しないで報告する

## ターミナル

- プロセスをkillするときは事前確認する

# アーキテクチャ

- **`ランタイム`**: Go 1.25
- **`データベース`**: PostgreSQL 18
- **`キャッシュ`**: Redis
- **`API`**: OpenAPI 3.0 / oapi-codegen
- **`テスト`**: Go testing / GoMock
- **`マイグレーション`**: golang-migrate v4
- **`Lint/Format`**: golangci-lint / gofmt

# ディレクトリ構造

- **`.devcontainer/`**: DevContainer（Docker + VS Code）用の開発環境定義
- **`cmd/app/`**: メインアプリケーション
- **`cmd/admin/`**: 管理者ツール
- **`cmd/migrator/`**: データ移行実行ツール
- **`cmd/fetcher/`**: データ取得バッチ
- **`cmd/updater/`**: データ更新バッチ
- **`docs/`**: アーキテクチャ・仕様・実装計画などのドキュメント
- **`internal/domain/`**: ドメインモデル・エンティティ定義
- **`internal/infra/handler/`**: HTTPハンドラー
- **`internal/infra/postgres/`**: PostgreSQLリポジトリ実装
- **`internal/infra/redis/`**: Redisキャッシュ実装
- **`internal/infra/external/`**: 外部API連携
- **`internal/pkg/`**: アプリケーション共通ユーティリティ
- **`internal/platform/`**: ログ・キャッシュ・DB・設定などのプラットフォーム機能
- **`internal/usecase/`**: ビジネスロジック・オーケストレーション層
- **`migrations/`**: DBマイグレーション定義（SQL）
- **`scripts/`**: シェルスクリプト・初期化スクリプト
- **`compose.yaml`**: 本番用Docker Compose設定
- **`compose.dev.yaml`**: 開発用Docker Compose追加設定
- **`Dockerfile`**: 本番用イメージ定義
- **`Dockerfile.dev`**: 開発用イメージ定義
- **`Dockerfile.postgres`**: PostgreSQLカスタムイメージ定義
- **`Makefile`**: ビルド・実行・テストなどのコマンド定義
- **`go.mod`**: Go依存関係定義
- **`openapi.yaml`**: API仕様のOpenAPI定義（型生成の元）
- **`oapi-codegen.yaml`**: oapi-codegenの設定
