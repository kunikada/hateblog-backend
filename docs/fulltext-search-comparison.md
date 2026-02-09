# 全文検索方式の比較（現状: pg_bigm + LIKE）

## 前提

- 現状: PostgreSQL + `pg_bigm` + `LIKE '%...%'`
- 対象: タイトル・抜粋・タグ
- 言語: 日本語 / 英語
- データ規模: 200万件未満
- 課題: 検索レイテンシが気になる
- 方針: Elasticsearch は運用コストの理由で検討外

## 結論（先に要点）

- 第一候補は **PGroonga**。
- **ParadeDB (pg_search)** は将来の高度検索（BM25重み付け、集計、ハイブリッド検索）を見据える場合の候補。
- 現状規模（200万件未満）かつ運用コスト重視なら、まず PGroonga が最も現実的。

## 比較観点

- 日本語・英語混在テキストでの検索品質
- 部分一致 / AND・OR の実装容易性
- 既存 PostgreSQL 運用への馴染み
- 書き込み時の負荷と運用リスク
- 将来拡張（ランキング、集計、ベクトル検索）

## 候補1: 現状継続（pg_bigm + LIKE）

### 評価

- 良い点
- 実装が単純で既存コードの変更が少ない
- 追加コンポーネント不要
- 注意点
- `%keyword%` 前提だと条件次第で遅くなりやすい
- ランキングや高度なクエリ表現が弱い
- 英語の語形変化や日本語の語彙同義展開には弱い

### 向いているケース

- 検索機能を最小限で維持したい
- 品質要件より開発・運用コスト最小化を優先

## 候補2: PGroonga

### 評価

- 良い点
- PostgreSQL 拡張として導入でき、ETL不要
- 日本語を含む多言語全文検索に強い
- クエリ演算子が豊富で、AND/OR/NOT・前方一致・正規表現などを扱いやすい
- DB内で完結するため整合性設計がシンプル
- 注意点
- 拡張依存（バージョン互換、アップグレード手順）の運用知識が必要
- レプリケーション設計時は PGroonga 固有の設定確認が必要
- 書き込み性能はインデックス分の追加負荷が発生

### 向いているケース

- まず検索速度を改善したい
- 日本語品質を上げたい
- 既存 PostgreSQL 中心の運用を維持したい

## 候補3: ParadeDB (pg_search)

### 評価

- 良い点
- PostgreSQL 上で BM25 検索を利用可能
- トークナイザーやフィルタ設定の自由度が高い
- 将来のハイブリッド検索（`pgvector` 併用）まで見据えやすい
- 注意点
- PGroonga より導入・運用の学習コストが高め
- 日本語はトークナイザー選定が重要（設定次第で品質差が出る）
- 1テーブル1 BM25 インデックスなど設計制約を前提にスキーマ設計が必要

### 向いているケース

- 検索スコア制御・集計・高度検索を早めに取り込みたい
- 将来的にベクトル検索との統合も計画している

## 3案比較（今回要件ベース）

| 観点 | pg_bigm + LIKE | PGroonga | ParadeDB (pg_search) |
|---|---|---|---|
| 日本語検索品質 | 中 | 高 | 中〜高（設定依存） |
| 英語検索品質 | 中 | 高 | 高 |
| 部分一致 | 高 | 高 | 設計次第 |
| AND/OR表現力 | 低〜中 | 高 | 高 |
| 速度改善余地 | 低〜中 | 高 | 高 |
| 導入容易性 | 高 | 中 | 中 |
| 運用負荷 | 低 | 中 | 中〜高 |
| 将来拡張性 | 低 | 中〜高 | 高 |

## 推奨方針

### 推奨

- **短中期（今すぐの遅さ改善）**: PGroonga へ移行
- **中長期（検索基盤の高度化）**: 必要になった時点で ParadeDB を再評価

### 理由

- Elasticsearch を除外する条件下で、速度改善と運用コストのバランスが最も良い。
- 200万件未満なら、PostgreSQL 拡張方式で十分に戦える可能性が高い。
- 既存の DB 一体運用を崩さずに改善できる。

## 導入判断の実務ポイント

- 先に測る指標を固定する（P50/P95レイテンシ、QPS、更新遅延、CPU/IO）。
- 検索クエリの上位パターン（短語、日本語複合語、英語複数語）で比較する。
- 本番相当データで以下を比較する。
- 検索速度
- 更新コスト（INSERT/UPDATE時）
- インデックスサイズ
- 運用手順の複雑さ（障害復旧、拡張アップグレード）

## 最終提案

- 現時点の要件では **PGroonga を第一候補として PoC** を実施。
- PoCで目標（例: 現状比でP95を30%以上改善）を満たせば採用。
- 目標未達、またはスコアリング要件が強い場合に ParadeDB を第二候補として検証。

## 参考（公式ドキュメント）

- PGroonga: https://pgroonga.github.io/
- PGroonga（レプリケーション）: https://pgroonga.github.io/reference/replication.html
- PGroonga（CREATE INDEX）: https://pgroonga.github.io/reference/create-index-using-pgroonga.html
- ParadeDB Docs: https://docs.paradedb.com/documentation
- ParadeDB（Install Extension）: https://docs.paradedb.com/deploy/self-hosted/extension
- ParadeDB（Create BM25 Index）: https://docs.paradedb.com/documentation/indexing/create-index
- ParadeDB（Tokenizers）: https://docs.paradedb.com/documentation/tokenizers/overview
