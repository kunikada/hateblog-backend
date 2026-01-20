# Testing Strategy

## Overview

このバックエンドは**API中心のアーキテクチャ**であるため、テスト戦略もAPIテストを中心に据える。
コアロジック（domain層）には最低限のユニットテストを用意し、主な品質保証はAPIレベルの統合テストで行う。

## Quick Start

```bash
# 全テスト実行（Dev Container内で推奨）
# PostgreSQL/Redisは自動的に起動済み
make test

# カバレッジレポート生成
make cover

# 特定のパッケージのみテスト
go test ./internal/domain/... -v
```

**Dev Container**: PostgreSQLとRedisはdocker-composeで自動起動されます。テストは特別な設定なしで実行できます。

**ホストマシン**: PostgreSQLとRedisは Docker Compose で起動し、マイグレーションは自動実行されます：
```bash
# コンテナ起動（PostgreSQL・Redis・アプリケーションが起動し、マイグレーション自動実行）
docker compose up -d

# テスト実行（デフォルトは postgres:5432 に接続）
go test ./...

# ホストマシンの localhost:5432 を使う場合は環境変数で接続先を上書き
TEST_POSTGRES_URL="postgresql://hateblog:changeme@localhost:5432/hateblog?sslmode=disable" go test ./...
```

### PostgreSQL統合テストについて

現在、以下のテストがPostgreSQL統合テストとして実装されています：

- `internal/infra/postgres/entry_repository_test.go` - エントリーリポジトリの全CRUD操作
- `internal/infra/postgres/tag_repository_test.go` - タグリポジトリの全操作

Dev Container内ではデフォルトで `postgres:5432` に接続します。ホストマシンで実行する場合は `TEST_POSTGRES_URL` 環境変数で接続先を指定してください。

## Testing Pyramid

```
        ┌─────────────┐
        │  E2E Tests  │  少数（主要フロー確認程度）
        └─────────────┘
       ┌───────────────┐
       │  API Tests    │  中心：各エンドポイントの動作を保証
       └───────────────┘
      ┌─────────────────┐
      │  Unit Tests     │  最低限：domainロジック・重要な関数のみ
      └─────────────────┘
```

### 1. API Tests（中心）

**目的**: 各APIエンドポイントが仕様通りに動作することを保証する

**対象**:
- REST API の全エンドポイント
- 正常系・異常系（バリデーション、認証・認可エラー等）
- OpenAPI仕様との整合性

**ツール・手法**:
- Docker ComposeでPostgreSQL/Redisが起動済み（Dev Container）
- 実際のHTTPリクエストを発行してレスポンスを検証
- `github.com/stretchr/testify/assert` でアサーション
- テストデータは各テストで prepare/cleanup（トランザクションロールバックまたは明示的削除）

**例**:
```go
func TestCreateUser_API(t *testing.T) {
    // testcontainers で DB/Redis 起動
    ctx := context.Background()
    pgContainer := startPostgresContainer(t, ctx)
    defer pgContainer.Terminate(ctx)

    // アプリケーション起動
    app := setupTestApp(t, pgContainer.ConnectionString())

    // HTTP リクエスト実行
    req := httptest.NewRequest("POST", "/api/v1/users", strings.NewReader(`{"name":"Alice"}`))
    rec := httptest.NewRecorder()
    app.ServeHTTP(rec, req)

    // レスポンス検証
    assert.Equal(t, http.StatusCreated, rec.Code)
    // JSON スキーマ検証や詳細なフィールドチェック
}
```

**カバレッジ目標**:
- 全エンドポイントの正常系: 100%
- 主要な異常系（400, 401, 403, 404, 500）: 80%以上

---

### 2. Unit Tests（最低限）

**目的**: コアビジネスロジックの正しさを保証する

**対象**:
- `domain/` 配下の Entity / ValueObject のバリデーション・変換ロジック
- 複雑な計算・判定ロジック（例: 料金計算、状態遷移、権限判定）
- ユーティリティ関数

**ツール・手法**:
- テーブル駆動テスト（複数のインプット・期待値を列挙）
- モック不要（domain は純粋関数を目指す）
- 必要に応じて `go.uber.org/mock` でリポジトリインターフェースをモック

**例**:
```go
func TestUser_Validate(t *testing.T) {
    tests := []struct {
        name    string
        user    domain.User
        wantErr bool
    }{
        {"valid", domain.User{Name: "Alice", Email: "alice@example.com"}, false},
        {"empty name", domain.User{Name: "", Email: "alice@example.com"}, true},
        {"invalid email", domain.User{Name: "Alice", Email: "invalid"}, true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.user.Validate()
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

**カバレッジ目標**:
- domain層の重要ロジック: 70%以上
- それ以外: カバレッジを追わない（APIテストで間接的にカバー）

---

## Manual API Testing

開発中の手動APIテスト手順。

### 前提
- Dev Container起動済み（またはホストマシンで `docker compose up -d` 実行済み）
- PostgreSQL/Redis が起動済み
- PostgreSQL マイグレーションが完了済み（自動実行）
- `.env.example` を `.env` にコピー済み（`APP_API_KEY_REQUIRED=false` がデフォルト）

### 手順

1. **アプリケーション起動**
   ```bash
   # Dev Container 内または docker compose で既に起動している場合はスキップ
   go run ./cmd/app/main.go
   ```

2. **テストデータ投入** (オプション)
   ```bash
   ./scripts/seed_manual.sh
   ```
   - 環境変数で接続先を上書き可能: `POSTGRES_HOST`, `POSTGRES_PORT`, `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`, `SQL_FILE`, `REDIS_HOST`, `REDIS_PORT`
   - **注意**: マイグレーションが既に実行されていることが前提（compose.yaml で自動実行）
   - **重要**: データ投入後、自動的にRedisキャッシュもクリアされます（古いデータが表示される問題を回避）

3. APIを叩く例
   - 新着:
     ```bash
     curl "http://localhost:8080/entries/new?date=20241202&min_users=5&limit=20"
     ```
   - 人気順:
     ```bash
     curl "http://localhost:8080/entries/hot?date=20241202&min_users=10"
     ```
   - タグ別:
     ```bash
     curl "http://localhost:8080/tags/go/entries?limit=20"
     ```
   - クリック記録（必要に応じて）:
     ```bash
     curl -X POST "http://localhost:8080/metrics/clicks" \
       -H "Content-Type: application/json" \
       -d '{"entry_id":"aaaaaaa1-aaaa-aaaa-aaaa-aaaaaaaaaaa1"}'
     ```

**Note**: データをリセットしたい場合は再度 `./scripts/seed_manual.sh` を実行してください（自動的にキャッシュもクリアされます）。

**キャッシュのみクリアしたい場合**:
```bash
./scripts/clear_cache.sh
```
または
```bash
redis-cli -h redis -p 6379 FLUSHDB
```

---

### 3. E2E Tests（少数）

**目的**: 主要なユーザーシナリオがエンドツーエンドで動作することを確認

**対象**:
- 代表的なフロー（例: ユーザー登録→ログイン→リソース作成→取得）
- クリティカルパス（課金処理、重要な状態遷移など）

**ツール・手法**:
- CI環境で実際のコンテナ構成（PostgreSQL, Redis, アプリ）を起動
- スクリプトまたは軽量なE2Eテストツール（例: `httpie`, `curl`, または Go の E2E テストスイート）

**カバレッジ目標**:
- 主要フロー 3-5 シナリオ程度

---

## Test Data Management

- **Fixtures**: 最小限の静的データを `testdata/` 配下にJSON/SQLで用意
- **Builder Pattern**: テストデータ生成用のビルダー関数を用意（例: `NewTestUser(t, opts...)`)
- **Cleanup**: 各テスト後にトランザクションロールバック、または明示的にDELETE

---

## CI/CD Integration と デプロイ

### デプロイフロー

**本番環境への展開**:
```bash
# VPS サーバー上
cd /opt/hateblog
git pull origin main
docker compose up -d --build
```

**起動時の自動処理**:
1. PostgreSQL コンテナが起動
2. Redis コンテナが起動
3. アプリケーションコンテナが起動（ヘルスチェック完了まで待機）
4. PostgreSQL初期化時に migrations ディレクトリのSQLファイルが自動実行
   - 000001_create_entries.up.sql
   - 000002_create_tags.up.sql
   - ...
5. アプリケーション起動完了

**データベース初期化の仕組み**:
- PostgreSQL公式イメージは `/docker-entrypoint-initdb.d/` にマウントされたSQLファイルを自動実行
- `compose.yaml` で `./migrations:/docker-entrypoint-initdb.d:ro` を指定
- ファイル名がアルファベット順に実行される（000001... 000002... など）

詳細は [DEPLOYMENT.md](./deployment.md) を参照

### ローカルテストの流れ

ローカル開発時は以下の流れで動作確認を行う：

1. Dev Container 内で `make test` を実行
2. テスト成功後、git push
3. VPS で `git pull && docker compose up -d --build` を実行

**注意**: GitHub Actions などの CI/CD ツールは使用していません。ローカルテストで十分な品質確保を行います。

---

## Mocking Strategy

- **原則**: モックは最小限に留める
- **infra層**: リポジトリインターフェースは必要に応じて `go.uber.org/mock` でモック生成
- **app層**: APIテストで実際のDBを使うため、基本的にモック不要
- **外部API**: HTTPクライアントのインターフェースをモック、または `httptest` でスタブサーバーを起動

**例**:
```go
//go:generate mockgen -source=user_repository.go -destination=mock_user_repository.go -package=domain
type UserRepository interface {
    Create(ctx context.Context, user *User) error
    FindByID(ctx context.Context, id string) (*User, error)
}
```

---

## Performance & Load Testing

- **ツール**: `k6` (https://k6.io/) または `vegeta`
- **目標**: p99 latency < 200ms, throughput > 1000 req/s (主要エンドポイント)
- **実施タイミング**: リリース前、または定期的（月次）
- **シナリオ**: 実運用を模したリクエストパターン（読み書き比率、同時接続数など）

---

## Contract Testing (将来検討)

- フロントエンドやモバイルアプリとのAPI契約を保証する場合、**Pact** などのコントラクトテストを導入
- OpenAPI仕様を Single Source of Truth として、スキーマ検証を自動化

---

## Testing Best Practices

1. **Fail Fast**: エラーは早期に検出できる構造にする（起動時のバリデーション等）
2. **Isolation**: 各テストは他のテストに依存しない（並列実行可能）
3. **Readability**: テストコードも本番コードと同等に可読性を重視
4. **Table-Driven**: 複数のケースを網羅的にテストする際はテーブル駆動テストを使う
5. **Minimize Mocks**: 実際のDBやインフラを使うことで、テストの信頼性を高める
6. **Don't Test Framework**: ライブラリ自体の動作はテストしない（自分のコードのみ）

---

## References

- [Testing in Go](https://go.dev/doc/tutorial/add-a-test)
- [Testcontainers for Go](https://golang.testcontainers.org/)
- [Testify](https://github.com/stretchr/testify)
- [Go Testing Best Practices](https://github.com/golang-standards/project-layout/blob/master/README.md)
