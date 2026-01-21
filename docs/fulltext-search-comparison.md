# 全文検索実装の比較

はてブログバックエンドにおける全文検索機能の実装方式を比較検討する。

## 検索要件

- **検索対象**: エントリーのタイトル、抜粋、タグ
- **データ規模**: 数十万〜数百万エントリー（想定）
- **言語**: 日本語（形態素解析が必要）
- **更新頻度**: 新着エントリーが継続的に追加される
- **検索機能**: キーワード検索、AND/OR検索、部分一致
- **運用環境**: VPS（メモリ・CPU制約あり）

## 選択肢1: PostgreSQL 全文検索

### 概要

PostgreSQL組み込みの全文検索機能（`tsvector`, `tsquery`）を使用。

### メリット

#### 1. インフラがシンプル
- **追加サービス不要**: PostgreSQLだけで完結
- **運用コスト低**: メンテナンス対象が増えない
- **リソース効率**: メモリ・CPU使用量が少ない

#### 2. データ整合性が高い
- **トランザクション対応**: DBトランザクション内で検索インデックス更新
- **同期の遅延なし**: INSERT/UPDATEと同時にインデックス更新
- **結合が容易**: SQLのJOINで他のテーブルと簡単に結合

#### 3. 導入が容易
- **学習コスト低**: SQL知識だけで実装可能
- **既存の知見活用**: チームのPostgreSQLスキルをそのまま活用
- **デプロイ不要**: compose.yamlに追加サービス記述不要

#### 4. コストが低い
- **VPSリソース**: PostgreSQL分のメモリ・CPUのみ
- **マネージドサービス**: 不要（PostgreSQLのみ）

### デメリット

#### 1. 日本語検索の精度
- **形態素解析が弱い**: デフォルトは単純な文字列分割
- **拡張必要**: `pg_bigm` や `Textsearch Japanese` などの拡張が必要
- **設定が複雑**: 日本語辞書の設定・メンテナンスが必要

#### 2. 検索機能の制約
- **あいまい検索**: ファジー検索やtypo toleranceが弱い
- **ランキング**: 検索スコアリングの柔軟性が低い
- **ハイライト**: 検索結果のハイライト表示が限定的

#### 3. スケーラビリティ
- **レプリケーション**: Read Replica でスケールは可能だが複雑
- **シャーディング**: 水平分割は困難
- **大規模データ**: 数千万件規模になると性能劣化

#### 4. 分析機能
- **アナリティクス**: 検索ログ分析、サジェストなどの機能が限定的

### 実装例

```sql
-- tsvector カラムを追加
ALTER TABLE entries ADD COLUMN search_vector tsvector;

-- GINインデックス作成
CREATE INDEX entries_search_idx ON entries USING GIN(search_vector);

-- 自動更新トリガー
CREATE TRIGGER entries_search_update
BEFORE INSERT OR UPDATE ON entries
FOR EACH ROW EXECUTE FUNCTION
tsvector_update_trigger(search_vector, 'pg_catalog.simple', title, excerpt);

-- 検索クエリ
SELECT * FROM entries
WHERE search_vector @@ to_tsquery('simple', 'keyword');
```

日本語対応（pg_bigm使用）:
```sql
CREATE EXTENSION pg_bigm;
ALTER TABLE entries
ADD COLUMN search_text TEXT;
CREATE INDEX entries_search_text_bigm_idx ON entries USING gin (search_text gin_bigm_ops);
SELECT * FROM entries WHERE search_text LIKE '%キーワード%';
```

### 推奨ライブラリ（Go）
- 標準の `pgx/v5` でSQL直接実行
- または `pg_bigm` + LIKE検索

---

## 選択肢2: Elasticsearch

### 概要

専用の全文検索エンジン Elasticsearch をサービスとして追加。

### メリット

#### 1. 検索機能が強力
- **高精度な日本語検索**: `kuromoji` 形態素解析器が標準サポート
- **あいまい検索**: Fuzzy search、Typo tolerance が強力
- **ランキング**: BM25などの高度なスコアリングアルゴリズム
- **ハイライト**: 検索結果のハイライト表示が簡単

#### 2. スケーラビリティが高い
- **水平スケール**: ノード追加で簡単にスケールアウト
- **シャーディング**: 自動で大規模データを分散
- **レプリケーション**: 高可用性構成が容易

#### 3. 分析機能が豊富
- **アグリゲーション**: 集計・分析クエリが強力
- **サジェスト**: オートコンプリート機能が簡単
- **検索ログ分析**: Kibanaで可視化

#### 4. 検索特化
- **パフォーマンス**: 全文検索に最適化された設計
- **柔軟性**: 検索要件の変更に対応しやすい

### デメリット

#### 1. インフラが複雑
- **追加サービス**: Elasticsearch サービスが必要
- **メモリ消費**: 最低2GB、推奨4GB以上のヒープメモリ
- **CPU消費**: インデックス更新時の負荷が高い
- **ディスク消費**: データの重複保存でストレージ使用量増加

#### 2. 運用コストが高い
- **学習コスト**: Elasticsearch固有の知識が必要
- **監視**: 専用の監視・アラート設定が必要
- **バージョンアップ**: メジャーバージョンアップ時の移行が複雑
- **VPSリソース**: 現在の構成に+2-4GBメモリ必要

#### 3. データ整合性の課題
- **非同期更新**: PostgreSQLとElasticsearchの間にタイムラグ
- **二重管理**: データの整合性を保つための仕組みが必要
- **障害時の挙動**: Elasticsearch障害時の代替検索が必要

#### 4. コスト増加
- **VPSサイズアップ**: メモリ増強が必要（例: 2GB → 8GB）
- **マネージドサービス**: Elastic Cloud等の利用でコスト増
- **開発工数**: 導入・運用の工数増加

### 実装例

```yaml
# compose.yaml に追加
services:
  elasticsearch:
    image: docker.elastic.co/elasticsearch/elasticsearch:8.15.0
    environment:
      - discovery.type=single-node
      - "ES_JAVA_OPTS=-Xms2g -Xmx2g"
      - xpack.security.enabled=false
    ports:
      - "127.0.0.1:9200:9200"
    volumes:
      - es_data:/usr/share/elasticsearch/data
    networks:
      - hateblog-network

volumes:
  es_data:
```

```go
// Go実装例
import "github.com/elastic/go-elasticsearch/v8"

// インデックス作成
PUT /entries
{
  "mappings": {
    "properties": {
      "title": {
        "type": "text",
        "analyzer": "kuromoji"
      },
      "excerpt": {
        "type": "text",
        "analyzer": "kuromoji"
      }
    }
  }
}

// 検索クエリ
POST /entries/_search
{
  "query": {
    "multi_match": {
      "query": "キーワード",
      "fields": ["title^2", "excerpt"]
    }
  },
  "highlight": {
    "fields": {
      "title": {},
      "excerpt": {}
    }
  }
}
```

### 推奨ライブラリ（Go）
- `github.com/elastic/go-elasticsearch/v8`
- `github.com/olivere/elastic/v7`

---

## 比較表

| 項目 | PostgreSQL全文検索 | Elasticsearch |
|------|-------------------|---------------|
| **導入の容易さ** | ⭐️⭐️⭐️⭐️⭐️ 非常に簡単 | ⭐️⭐️⭐️ 追加サービス必要 |
| **日本語検索精度** | ⭐️⭐️⭐️ pg_bigm で対応可能 | ⭐️⭐️⭐️⭐️⭐️ kuromoji で高精度 |
| **検索機能の豊富さ** | ⭐️⭐️⭐️ 基本的な機能 | ⭐️⭐️⭐️⭐️⭐️ 非常に豊富 |
| **スケーラビリティ** | ⭐️⭐️⭐️ Read Replica まで | ⭐️⭐️⭐️⭐️⭐️ 水平スケール容易 |
| **運用コスト** | ⭐️⭐️⭐️⭐️⭐️ 非常に低い | ⭐️⭐️ 高い |
| **リソース消費** | ⭐️⭐️⭐️⭐️⭐️ 少ない | ⭐️⭐️ 多い（+2-4GB） |
| **データ整合性** | ⭐️⭐️⭐️⭐️⭐️ トランザクション保証 | ⭐️⭐️⭐️ 非同期で遅延 |
| **学習コスト** | ⭐️⭐️⭐️⭐️⭐️ SQL知識のみ | ⭐️⭐️ ES固有の知識必要 |

---

## 推奨事項

### プロジェクトの現状を考慮した推奨

#### **推奨: PostgreSQL全文検索（pg_bigm）**

**理由**:

1. **初期段階**: まだMVP構築フェーズ。複雑な検索機能より、まず動くものを作ることが優先
2. **リソース制約**: VPS環境で追加メモリ2-4GBの確保が困難な可能性
3. **運用負荷**: 小規模チーム想定で、運用対象を増やすべきではない
4. **データ規模**: 初期は数万〜数十万件規模で、PostgreSQLで十分対応可能
5. **要件**: 基本的なキーワード検索で、高度なランキングやサジェストは必須ではない

**実装方針**:
```sql
-- pg_bigm を使用したシンプルな実装
CREATE EXTENSION pg_bigm;
ALTER TABLE entries
ADD COLUMN search_text TEXT;
CREATE INDEX entries_search_text_bigm ON entries USING gin (search_text gin_bigm_ops);

-- LIKEクエリで検索（pg_bigmがインデックスを使用）
SELECT * FROM entries
WHERE search_text LIKE '%キーワード%'
ORDER BY bookmark_count DESC
LIMIT 50;
```

### Elasticsearchへの移行タイミング

以下の条件を**2つ以上**満たした場合、Elasticsearchへの移行を検討：

1. **データ規模**: エントリー数が100万件を超える
2. **検索精度の要求**: ユーザーから「検索結果が不正確」というフィードバックが多い
3. **高度な機能要求**: サジェスト、ファセット検索、検索ログ分析が必要になった
4. **パフォーマンス問題**: PostgreSQL検索が遅くなり、クエリ最適化でも改善しない
5. **リソース確保**: VPSのメモリを8GB以上に増強できる

### 段階的な実装戦略

**Phase 1（現在）**: PostgreSQL + LIKE検索
- 最もシンプル、すぐに実装可能
- pg_bigm なしでも動作する

**Phase 2（データ増加時）**: PostgreSQL + pg_bigm
- pg_bigm 拡張を導入
- GINインデックスで高速化
- 日本語検索精度向上

**Phase 3（必要になったら）**: Elasticsearch移行
- データ規模・要件が拡大した段階で検討
- PostgreSQLからの移行スクリプト作成
- 段階的にカットオーバー

---

## 結論

**PostgreSQL全文検索（pg_bigm）を推奨**

- 初期段階では十分な機能とパフォーマンス
- 運用コストとリソース消費が最小
- 将来的にElasticsearchへの移行も可能

Elasticsearchは、サービスが成長し、高度な検索機能が必要になった段階で検討する。
