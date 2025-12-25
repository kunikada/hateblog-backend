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

**ホストマシン**: PostgreSQLとRedisを手動で起動する必要があります：
```bash
# PostgreSQLとRedisを起動
docker compose up -d postgres redis

# マイグレーション実行
DB_URL="postgresql://hateblog:changeme@localhost:5432/hateblog?sslmode=disable" make migrate-up

# localhostを使う場合は環境変数で接続先を上書き
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
- Dev Container起動済み
- PostgreSQL/Redisは自動起動済み
- `.env.example` を `.env` にコピー済み（`APP_API_KEY_REQUIRED=false` がデフォルト）

### 手順
1. Dev Container内でアプリケーションを起動
   ```bash
   go run ./cmd/app/main.go
   ```

2. テストデータ投入（2024-12-01/02 のエントリー4件を投入）
   ```bash
   ./scripts/seed_manual.sh
   ```
   - 環境変数で上書き可能: `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `POSTGRES_SERVICE`, `SQL_FILE`

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

**Note**: データをリセットしたい場合は再度 `./scripts/seed_manual.sh` を実行してください。

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

## CI/CD Integration

### CI Pipeline
```yaml
test:
  - go test ./...                    # ユニットテスト
  - go test -tags=integration ./...  # APIテスト（testcontainers 使用）
  - golangci-lint run                # 静的解析
  - govulncheck ./...                # 脆弱性チェック
```

### CD Pipeline
- E2Eテストをステージング環境で実行
- 成功後にプロダクション環境へデプロイ

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
