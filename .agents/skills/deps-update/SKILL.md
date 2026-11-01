---
name: deps-update
description: このリポジトリの Go 依存関係の更新候補を確認し、必要に応じて更新する。ユーザーが古い依存の確認や依存更新を求めたときに使う。
---

# Dependencies Update

## 方針

- 標準ライブラリ優先。外部依存は厳選・少数・更新容易なものに限る。
- 依存更新は Renovate の PR を起点に行う。手動更新は例外扱い。
- 破壊的変更のある lib は `internal/adapter` で吸収する。

## 手順

1. `make deps-outdated` で更新候補を一覧表示する（`go list -m -u all` を解析）。
2. Renovate の PR がある場合は、更新理由・影響・リスクを確認し、必要なら `go test ./...` を実行する。
3. 手動更新が必要な場合のみ、ユーザーに確認を取ってから `go get <module>@<version>` と `go mod tidy` を実行する。
4. 結果（更新したモジュール、バージョン、影響）を報告する。
