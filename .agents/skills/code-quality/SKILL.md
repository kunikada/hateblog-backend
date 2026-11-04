---
name: code-quality
description: このリポジトリの Go 用フォーマット・lint・テスト・カバレッジを make タスクで実行する。ユーザーが fmt / lint / test / cover を求めたときに使う。依存関係の更新は deps-update スキルを使う。
---

# Code Quality

必須ツール: `gofmt` / `goimports`、`go vet`、`golangci-lint`（`staticcheck` 含む）。
コマンドは直接叩かず、`make` タスクで統一する。

## タスク

| 目的 | コマンド | 内容 |
| --- | --- | --- |
| フォーマット | `make fmt` | `gofmt -s -w .` + `goimports -w -local hateblog .` |
| lint | `make lint` | `golangci-lint run --config .golangci.yml` + `make depguard`（依存境界チェック） |
| テスト | `make test` | `go test ./... -v -race -shuffle=on -timeout=5m` |
| カバレッジ | `make cover` | `cover.out` / `cover.html` を生成 |

## 手順

1. ユーザーが求めたタスクだけを実行する（AGENTS.md により、実装後に自動で実行しない）。
2. 複数実行する場合は `fmt` → `lint` → `test` の順。
3. 失敗時は出力を要約して報告し、修正に入る前に確認を取る。
4. `make cover` で生成された `cover.out` / `cover.html` は、ユーザーが不要とすれば削除する。

## PR 前チェック

`make fmt lint test` がすべて通ること。
