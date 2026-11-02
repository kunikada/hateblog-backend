---
name: deps-update
description: このリポジトリの Go 依存関係・Go 本体・コンテナイメージの更新候補を確認し、必要に応じて更新する。ユーザーが古い依存の確認や依存更新、Go やイメージのバージョンアップを求めたときに使う。
---

# Dependencies Update

## 方針

- 標準ライブラリ優先。外部依存は厳選・少数・更新容易なものに限る。
- 依存更新は Renovate の PR を起点に行う。手動更新は例外扱い。
- 破壊的変更のある外部 lib は `internal/infra/external` で吸収する。
- 更新の実行（`go get` / 編集）前に、対象とバージョンをユーザーに確認する。

## Renovate の対象範囲

- 対象: `go.mod` のモジュール（`go mod tidy` 込み）、Dockerfile / compose のイメージタグ。
- 対象外（手動対応が必要）:
  - `Dockerfile.postgres` の `PG_BIGM_REF` / `PG_BIGM_SHA256`
  - Debian 系統変更に伴うパッケージ名修正
  - バージョンを記載したドキュメント

## 手順: Go モジュール

1. `make deps-outdated` で更新候補を一覧表示する（`go list -m -u all` を解析）。
2. メジャーバージョン（`/v2` 以降）は `go list -m -u` で検出されないため、主要な直接依存は個別に新メジャーの有無を確認する。
3. Renovate の PR がある場合は、更新理由・影響・リスク（CHANGELOG の破壊的変更）を確認する。
4. 手動更新は `go get <module>@<version>` → `go mod tidy` で行う。
   - `go run ./cmd/tools/depsoutdated -update` は indirect を含む全モジュールを一括更新するため、使う場合は事前に確認を取る。
5. 検証（ユーザーの指示がある場合に実行、なければ推奨として報告）:
   - `go mod verify`
   - `make lint` / `make test` / `make build`
   - `make security`（gosec + govulncheck）
   - `go.mod` / `go.sum` の差分確認

## 手順: Go 本体

Go のバージョンを上げる場合は以下をまとめて揃える。

- `go.mod` の `go` ディレクティブ
- `Dockerfile` / `Dockerfile.dev` の `golang` イメージタグ
- `.golangci.yml` の `run.go`
- `AGENTS.md` / `AGENTS.ja.md` の Runtime、`README.md` の技術スタック、`docs/architecture.md` の Runtime / Stack

## 手順: コンテナイメージ

- 対象: `golang`、`distroless`（`Dockerfile`）、`postgres`（`Dockerfile.postgres`）、`redis`（`compose.yaml`）。
- Debian 系統（例: bookworm → trixie）を変える場合:
  - `golang` / `postgres` のタグと `distroless/static-debianNN` を揃える。
  - `Dockerfile.postgres` の `apt-mark auto` 一覧を新系統のパッケージ名に合わせる。実際の依存は次で確認する:
    `docker run --rm postgres:<tag> bash -c 'apt-get update -qq && apt-get install -s --no-install-recommends build-essential postgresql-server-dev-<major> ca-certificates curl locales | grep ^Inst'`
  - glibc 更新で照合順序が変わる可能性があるため、既存 DB ボリュームでは `REINDEX` を案内する。
- pg_bigm の新リリースを確認し、更新時は `PG_BIGM_REF` と `PG_BIGM_SHA256` を合わせて変更する。
- PostgreSQL / Redis のバージョン記載（`README.md`、`docs/architecture.md`）を揃える。

## 報告

更新したモジュール・イメージ、旧→新バージョン、影響、未実施の検証を報告する。
