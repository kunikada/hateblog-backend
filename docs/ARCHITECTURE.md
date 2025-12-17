# Architecture

## Core Principles
1) Simplicity over Cleverness — 便利トリックより直交性・明示性
2) Readability First — 防御レイヤーを積むより読みやすく
3) Small Interfaces — 使う側が必要な面だけを定義
4) Backward Compatibility — 破壊は慎重、移行パスを用意

## Runtime / Stack
- Go 1.25.x（CI は 1.25 / 1.24 の2系を回す）
- PostgreSQL 18+
- Redis 8.4+（キャッシュ）
- OpenAPI 3.0 仕様で REST を定義

## Libraries / Tools
- HTTP ルーター: `github.com/go-chi/chi/v5`
- OpenAPI サーバー実装: `github.com/oapi-codegen/oapi-codegen/v2`
- DB ドライバ: `github.com/jackc/pgx/v5`（`pgxpool` 利用）
- マイグレーション: `github.com/golang-migrate/migrate/v4`
- キャッシュ: `github.com/redis/go-redis/v9`
- Config ロード: `github.com/caarlos0/env/v10`
- ロギング: 標準 `log/slog`（開発時は `github.com/lmittmann/tint` で整形）
- Observability: メトリクスは Prometheus (`github.com/prometheus/client_golang/prometheus/promhttp`)。トレースは段階導入（初期は省略し、必要になったら `go.opentelemetry.io/otel` を追加）。
- テスト: `github.com/stretchr/testify`、統合テスト用に `github.com/testcontainers/testcontainers-go`
- モック生成: 標準 `go test` の `-run` に依存しない場合は `go.uber.org/mock` を使用可
- Lint/Format: `golangci-lint`, `gofmt`, `goimports`（CONTRIBUTING に準拠）
- セキュリティ: `govulncheck` に加え、`gosec`（静的セキュリティリンター）、`aquasecurity/trivy`（コンテナイメージスキャン）
- 設計ノート:
  - oapi-codegen/v2: OpenAPI-first を強制できる一方、生成コードのカスタマイズ性とレビュー負荷に注意。生成物はリポジトリに含め、レビュー手順を明示し、必要ならカスタムテンプレートを用意。
  - caarlos0/env/v10: 環境変数のみなら適。設定ファイル併用が必要なら `spf13/viper` 等へ切替を検討。
  - testify: `require` など最小限のユーティリティ使用にとどめる。`suite` は避ける。モックは `go.uber.org/mock` を優先。
  - Redis 永続化: データは PostgreSQL から再構築可能とし、RDB/AOF を無効化してパフォーマンス最優先で運用。
  - バックアップ: PostgreSQL 18 は物理/論理バックアップを必須。`pgBackRest` または `wal-g` を採用し、スナップショット＋WAL でポイントインタイムリカバリを担保。
  - マイグレーション: `golang-migrate` の up/down を用意。down は安全な範囲に限定し、本番ロールバック手順（例: 直前バージョンの down 適用＋アプリロールバック）を運用 Runbook に明示。

## Module / Layout
```
.
├─ cmd/<app>          # main: DIとwire-upだけ（ロジック禁止）
├─ internal/
│  ├─ domain/         # Entity/ValueObject/Repository interface（境界の中心）
│  ├─ usecase/        # UseCase/Service（ビジネス操作の調停）
│  ├─ infra/          # DB/HTTP/外部APIの実装（adapter）
│  └─ platform/       # ログ/設定/観測基盤など
└─ pkg/               # 外部にも使える純粋Libがある時のみ
```

## Layering / Rules
- 物理境界はディレクトリで分離し、依存方向は usecase → domain（interface） ← infra に固定。
- domain は純粋に保ち、他層への import を禁止（depguard でCIチェック）。外部ライブラリ依存も最小限にする。
- interface は利用側（呼び出す層）に置く。実装は提供側が持ち、依存を逆転させる。
- main/cmd は配線専用。起動設定・DI・handler登録のみを行い、ビジネスロジックを置かない。
- リファクタリングは小さく・頻繁に実施し、境界崩れを早期に検知・修正する。

## Data Flow (依存方向)
UI/Handler → usecase → domain(interface) ← infra(impl)

- domain は純粋（副作用なし）を目指す
- infra は標準 lib 優先（`net/http`, `database/sql`, `encoding/*` …）

## Configuration
- `env`/フラグで注入、構造体にバインド。デフォルト安全値。
- 機密は環境変数 or 外部秘密管理。設定値は起動時に検証。

## Error & Observability
- error は境界でラップ。HTTP/CLI は一カ所でマッピング。
- ログは構造化。トレースは OpenTelemetry、メトリクスは Prometheus。
- SLA/SLI を明確化（p99 latency / error rate / 依存先の可用性）。

## Concurrency & Resource
- 1リクエスト＝1コンテキスト。外部 I/O はタイムアウト必須。
- ワーカー数／チャネル容量は明示。無制限キューは禁止。

## Testing Strategy
このバックエンドは**API中心**のため、テストもAPIテストを中心に据える。
詳細は [TESTING.md](./TESTING.md) を参照。

**概要**:
- **API Tests（中心）**: 全エンドポイントの正常系・異常系を `testcontainers-go` で実環境に近い形でテスト
- **Unit Tests（最低限）**: domain層のコアロジック・重要な関数のみテーブル駆動テストで保証
- **E2E Tests（少数）**: 主要なユーザーシナリオをエンドツーエンドで確認

## Versioning / Release
- セマンティックバージョニング。Breaking は `CHANGELOG.md`。
- マイグレーションは前方互換の二段階（書き→読み）を原則。

## Security Baseline
- 入力検証／コンテキスト期限／出力のJSON スキーマ固定。
- 依存性は `go mod tidy` と脆弱性チェック（`govulncheck`）をCIで。
