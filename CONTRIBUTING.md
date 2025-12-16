# Contributing

## Philosophy
- シンプル最優先。読みやすさ＞抽象の“賢さ”。暗黙を嫌い、明示する。
- 「小さく作り、小さくレビュー」。PR は 300 行目安、1 機能／1 修正。
- Go の流儀に合わせる（Effective Go / Go way）。独自流は避ける。

## Supported Go Versions
- `go 1.N`（最新+1つ前）をサポート。CI で両方回す。毎リリース追随。

## Toolchain & Lints
- 必須: `gofmt`/`goimports`、`go vet`、`golangci-lint`（`staticcheck` 含む）
- 推奨: `make` タスクで統一
  - `make fmt` → `gofmt -s -w ./` + `goimports -w ./`
  - `make lint` → `golangci-lint run` + `depguard` で依存境界チェック
  - `make test` → `go test ./... -race -shuffle=on`
  - `make cover` → `go test ./... -coverprofile=cover.out`
  - `make deps-outdated` → `go list -m -u all` を解析し、更新候補の依存を一覧表示

## Branch / Commit
- ブランチ: `feat/*`, `fix/*`, `chore/*`, `docs/*`
- Commit は要点を短く。スコープ明記。例: `feat(handler): add CreateUser`

## Code Style（抜粋）
- 命名: 公開 API は大文字開始、内部は短く意味明確に（Effective Go）
- 受け取り方:
  - メソッドレシーバは基本 `*T`。値コピーが明確な場合のみ `T`。  
  - インタフェースは **小さく**、**利用側が定義**（"accept interfaces, return concrete"）
- 構成: `cmd/`（エントリ）、`internal/`（非公開）、`pkg/`（公開ライブラリ的）、`internal/{app,domain,infra}` などに分割
- コンストラクタ: `NewXxx(...) (*Xxx, error)`。ゼロ値有用を意識。

## Error Handling
- 例外は使わない。**戻り値で返す**。`panic` は初期化致命エラーのみ。
- ラップ: `fmt.Errorf("context: %w", err)` で原因を保持。判定は `errors.Is/As`
- センチネル: `var ErrXxx = errors.New("...")` を必要最小限で
- メッセージは行動可能情報＋最小の文脈。多段ラップの重複は禁止。

## Logging
- 構造化ログ（`log/slog` or `zap`）。**人に見せる文**より**機械可読**を重視。
- ログレベル運用: default=INFO、ノイジー箇所は DEBUG に落とす。

## Context
- 第一引数に `ctx context.Context`。**保存しない／グローバルに置かない**。
- キャンセル／タイムアウトは呼び出し側で制御。I/O 境界で必ず伝播。

## Concurrency
- ゴルーチンを起動する前に **いつ止めるか** を決める（ctx か close）
- 共有状態の同期は基本 `sync.Mutex/atomic`。**チャネルはシグナル伝達中心**。
- バッファなしチャネルの無制限待ちや“送りっぱなし”は禁止（リーク注意）
- `-race` を常に通す。並行テストは `t.Parallel()` で最小単位に。

## Testing
- テーブル駆動＋サブテスト。I/O はインタフェース差し替えでフェイク化。
- エラー分岐を最優先で網羅。**成功ケースより先に失敗ケース**。
- ベンチ: `go test -bench . -benchmem`。効果が出ない最適化は入れない。
- カバレッジは閾値を設けないが、**クリティカル経路**は 100% を狙う。

## Dependencies
- 標準ライブラリ優先。外部は厳選・少数・更新容易。
- 破壊的変更のある lib は `internal/adapter` で吸収。
- 依存更新フロー:
  1. `make deps-outdated` で更新候補を確認
  2. `go get example.com/mod@vX.Y.Z` で個別に上げる（`go get -u ./...` は慎重に）
  3. `go mod tidy` ＋ `go test ./...` で差分と動作を確認
  4. `go.mod` / `go.sum` の diff が最小になっているかをチェック

## API/CLI 契約
- 入出力は明確・最小。**Breaking change は Release Note 必須**。

## PR Checklist
- [ ] `make fmt lint test` が通る
- [ ] public シンボルに Godoc がある
- [ ] 失敗ケースのテストがある
- [ ] 受け手が読めるエラーメッセージになっている
