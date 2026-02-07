# キャッシュ設計書

## 目的

データベース負荷を軽減し、API応答速度を向上させるため、各エンドポイントに適切なキャッシュ戦略を実装します。

## キャッシュ技術スタック

- **Redis**: 分散キャッシュストア
- **キャッシュキー形式**: `hateblog:{endpoint}:{params_hash}`
- **シリアライゼーション**: JSON
- **圧縮**: アプリケーション層で実装（詳細は後述）

## APIエンドポイント別キャッシュ設計

### 1. 新着順エントリー一覧 (`GET /entries/new`)

**キャッシュ戦略**: 時間ベースのTTL

- **キャッシュキー**: `hateblog:entries:new:{date}:{min_users}:{limit}:{offset}`
- **TTL**: 5分
- **理由**: 新着エントリーは頻繁に更新されるが、5分程度の遅延は許容可能
- **キャッシュ対象**: レスポンス全体（EntryListResponse）
- **DB負荷軽減効果**: 高（頻繁にアクセスされるエンドポイント）

**実装メモ**:
```go
cacheKey := fmt.Sprintf("hateblog:entries:new:%s:%d:%d:%d", date, minUsers, limit, offset)
ttl := 5 * time.Minute
```

**無効化条件**:
- エントリーの新規追加時（バッチ処理後）
- パターンマッチで `hateblog:entries:new:*` を削除

---

### 2. 人気順エントリー一覧 (`GET /entries/hot`)

**キャッシュ戦略**: 時間ベースのTTL

- **キャッシュキー**: `hateblog:entries:hot:{date}:{min_users}:{limit}:{offset}`
- **TTL**: 10分
- **理由**: ブックマーク数の変動は比較的緩やか
- **キャッシュ対象**: レスポンス全体（EntryListResponse）
- **DB負荷軽減効果**: 高（人気エンドポイント）

**実装メモ**:
```go
cacheKey := fmt.Sprintf("hateblog:entries:hot:%s:%d:%d:%d", date, minUsers, limit, offset)
ttl := 10 * time.Minute
```

**無効化条件**:
- ブックマーク数更新バッチ実行後
- パターンマッチで `hateblog:entries:hot:*` を削除

---

### 3. アーカイブデータ取得 (`GET /archive`)

**キャッシュ戦略**: 長期TTL（日次更新）

- **キャッシュキー**: `hateblog:archive:{year}:{month}:{min_users}`
- **TTL**: 24時間
- **理由**: 過去のアーカイブデータは不変
- **キャッシュ対象**: ArchiveResponse
- **DB負荷軽減効果**: 中（集計クエリが重い）

**実装メモ**:
```go
cacheKey := fmt.Sprintf("hateblog:archive:%d:%s:%d", year, monthStr, minUsers)
ttl := 24 * time.Hour
```

**無効化条件**:
- 日次バッチ実行後（新しいデータが追加された場合のみ）
- 当月分のみ無効化: `hateblog:archive:{current_year}:{current_month}:*`

---

### 4. 年次ランキング (`GET /rankings/yearly`)

**キャッシュ戦略**: 長期TTL（不変データ）

- **キャッシュキー**: `hateblog:rankings:yearly:{year}:{min_users}`
- **TTL**: 7日間（過去年は実質無期限）
- **理由**: 過去年のランキングは確定データ
- **キャッシュ対象**: RankingResponse
- **DB負荷軽減効果**: 高（重い集計クエリ）
- **キャッシュ対象条件**: `limit=100` かつ `offset=0` のときのみ
- **limit上限**: 100

**実装メモ**:
```go
cacheKey := fmt.Sprintf("hateblog:rankings:yearly:%d:%d", year, minUsers)
// 過去年は長期キャッシュ、当年は短期
if year < time.Now().Year() {
    ttl = 30 * 24 * time.Hour // 30日
} else {
    ttl = 1 * time.Hour // 当年は1時間
}
```

**無効化条件**:
- 当年のみ日次バッチで無効化

---

### 5. 月次ランキング (`GET /rankings/monthly`)

**キャッシュ戦略**: 時間ベースのTTL

- **キャッシュキー**: `hateblog:rankings:monthly:{year}:{month}:{min_users}`
- **TTL**: 1時間（当月）/ 24時間（過去月）
- **理由**: 過去月は確定、当月は更新中
- **キャッシュ対象**: RankingResponse
- **DB負荷軽減効果**: 高（重い集計クエリ）
- **キャッシュ対象条件**: `limit=100` かつ `offset=0` のときのみ
- **limit上限**: 100

**実装メモ**:
```go
cacheKey := fmt.Sprintf("hateblog:rankings:monthly:%d:%d:%d", year, month, minUsers)
now := time.Now()
if year == now.Year() && month == int(now.Month()) {
    ttl = 1 * time.Hour
} else {
    ttl = 24 * time.Hour
}
```

**無効化条件**:
- 当月のみ日次バッチで無効化

---

### 6. 週次ランキング (`GET /rankings/weekly`)

**キャッシュ戦略**: 時間ベースのTTL

- **キャッシュキー**: `hateblog:rankings:weekly:{year}:{week}:{min_users}`
- **TTL**: 30分（当週）/ 24時間（過去週）
- **理由**: 週の途中は頻繁に更新される
- **キャッシュ対象**: RankingResponse
- **DB負荷軽減効果**: 高（重い集計クエリ）
- **キャッシュ対象条件**: `limit=100` かつ `offset=0` のときのみ
- **limit上限**: 100

**実装メモ**:
```go
cacheKey := fmt.Sprintf("hateblog:rankings:weekly:%d:%d:%d", year, week, minUsers)
currentYear, currentWeek := time.Now().ISOWeek()
if year == currentYear && week == currentWeek {
    ttl = 30 * time.Minute
} else {
    ttl = 24 * time.Hour
}
```

**無効化条件**:
- 当週のみ日次バッチで無効化

---

### 7. タグ別エントリー一覧 (`GET /tags/entries/{tag}`)

**キャッシュ戦略**: 時間ベースのTTL（初回ページのみ）

- **キャッシュキー**: `hateblog:tags:{tag_name}:entries:{sort}:{min_users}:100:0`
- **TTL**: 10分
- **理由**: タグ別エントリーは中程度の更新頻度
- **キャッシュ対象**: EntryListResponse
- **DB負荷軽減効果**: 中〜高（人気タグは高負荷）
- **キャッシュ対象条件**: `limit=100` かつ `offset=0` のときのみ
- **limit上限**: 100

**実装メモ**:
```go
cacheKey := fmt.Sprintf("hateblog:tags:%s:entries:%s:%d:100:0",
    url.QueryEscape(tagName), sort, minUsers)
ttl := 10 * time.Minute
```

**最適化**:
- 人気タグTOP100はウォームアップキャッシュ（事前生成）
- タグ閲覧履歴（tag_view_history）を利用して優先度付け

**無効化条件**:
- タグ更新バッチ実行後
- 特定タグのみ無効化: `hateblog:tags:{tag_name}:*`

---

### 8. タグ一覧取得 (`GET /tags`)

**キャッシュ戦略**: 長期TTL

- **キャッシュキー**: `hateblog:tags:list:{limit}:{offset}`
- **TTL**: 1時間
- **理由**: タグマスタは比較的静的
- **キャッシュ対象**: TagListResponse
- **DB負荷軽減効果**: 中（ソート付き集計）

**実装メモ**:
```go
cacheKey := fmt.Sprintf("hateblog:tags:list:%d:%d", limit, offset)
ttl := 1 * time.Hour
```

**無効化条件**:
- 新規タグ追加時
- エントリー数更新バッチ後

---

### 9. キーワード検索 (`GET /search`)

**キャッシュ戦略**: 短期TTL（初回ページのみ）

- **キャッシュキー**: `hateblog:search:{query_hash}:{sort}:{min_users}:100:0`
- **TTL**: 15分
- **理由**: 検索結果は頻繁に変わる可能性がある
- **キャッシュ対象**: SearchResponse
- **DB負荷軽減効果**: 高（全文検索は重い）
- **キャッシュ対象条件**: `limit=100` かつ `offset=0` のときのみ
- **limit上限**: 100

**実装メモ**:
```go
queryHash := sha256Hash(query) // クエリをハッシュ化
cacheKey := fmt.Sprintf("hateblog:search:%s:%s:%d:100:0", queryHash, sort, minUsers)
ttl := 15 * time.Minute
```

**最適化**:
- 同一クエリの重複検索を防ぐ
- 人気検索クエリTOP50はウォームアップキャッシュ

**無効化条件**:
- 全文検索インデックス更新後
- パターンマッチで `hateblog:search:*` を削除

---

### 10. Faviconプロキシ (`GET /favicons`)

**キャッシュ戦略**: キャッシュ不要（設計済み）

- **理由**: このエンドポイントは既にFaviconをキャッシュする機能として設計されているため、追加のRedisキャッシュは不要
- **既存のキャッシュ**: Google Favicon API経由で取得後、アプリケーション層でキャッシュ済み

---

### 11. クリック計測 (`POST /metrics/clicks`)

**キャッシュ戦略**: キャッシュ不要（書き込み専用）

- **理由**: メトリクスデータは記録のみで読み取りなし
- **最適化**: 非同期書き込み + バッファリング
- **DB負荷軽減**: バッチ挿入で負荷分散

---

### 12. APIキー発行 (`POST /api-keys`)

**キャッシュ戦略**: キャッシュ不要

- **理由**: 低頻度のオペレーション
- **セキュリティ**: 認証情報はキャッシュしない

---

### 13. ヘルスチェック (`GET /health`)

**キャッシュ戦略**: キャッシュ不要

- **理由**: リアルタイムのヘルスステータスが必要

---

## キャッシュ無効化戦略

### バッチ処理と連動した無効化

| バッチ処理 | 無効化対象キャッシュ |
|-----------|-------------------|
| エントリー取得バッチ | `hateblog:entries:new:*`, `hateblog:entries:hot:*` |
| ブックマーク数更新 | `hateblog:entries:hot:*`, `hateblog:rankings:*` |
| タグ更新バッチ | `hateblog:tags:*` |
| 全文検索インデックス更新 | `hateblog:search:*` |
| 日次集計バッチ | `hateblog:archive:*`, `hateblog:rankings:*` |

### 部分無効化パターン

```go
// 特定日付のエントリーキャッシュのみ削除
redis.Del(ctx, fmt.Sprintf("hateblog:entries:new:%s:*", date))
redis.Del(ctx, fmt.Sprintf("hateblog:entries:hot:%s:*", date))

// 特定タグのキャッシュのみ削除
redis.Del(ctx, fmt.Sprintf("hateblog:tags:%s:*", tagName))
```

---

## キャッシュヒット率の監視

### メトリクス収集

```go
type CacheMetrics struct {
    Endpoint   string
    HitRate    float64
    Hits       int64
    Misses     int64
    AvgLatency time.Duration
}
```

### 監視対象

- キャッシュヒット率（目標: 80%以上）
- キャッシュミス時のDB負荷
- キャッシュサイズ（メモリ使用量）
- TTL期限切れの頻度

---

## キャッシュウォームアップ

### 事前キャッシュ生成

定期バッチで以下をプリロード:

1. **今日・昨日の人気エントリー** (min_users=10, 50, 100)
2. **人気タグTOP100のエントリー**
3. **直近1ヶ月のランキング**
4. **よく検索されるキーワードTOP50**

実行タイミング: 日次バッチ完了後

---

## パフォーマンス目標

| エンドポイント | キャッシュヒット時 | キャッシュミス時 | DB負荷軽減率 |
|--------------|-----------------|---------------|------------|
| /entries/new | < 10ms | < 100ms | 90% |
| /entries/hot | < 10ms | < 150ms | 90% |
| /rankings/* | < 10ms | < 500ms | 95% |
| /search | < 10ms | < 300ms | 85% |
| /tags/*/entries | < 10ms | < 150ms | 80% |
| /archive | < 10ms | < 200ms | 90% |

---

## 実装チェックリスト

- [ ] Redis接続プール設定
- [ ] キャッシュミドルウェア実装
- [ ] エンドポイント別TTL設定
- [ ] バッチ処理との無効化連携
- [ ] キャッシュメトリクス収集
- [ ] ウォームアップバッチ実装
- [ ] フォールバック処理（Redis障害時）
- [ ] キャッシュキー命名規則の統一
- [ ] ドキュメント更新（運用手順書）

---

## 注意事項

1. **Redis障害時の動作**: キャッシュ取得失敗時は自動的にDBクエリにフォールバック
2. **メモリ管理**: Redis最大メモリ設定 + LRU削除ポリシー
3. **セキュリティ**: 認証情報や個人情報はキャッシュしない
4. **整合性**: 重要な更新後は即座にキャッシュ無効化

---

## パラメータ別キャッシュ戦略

### 前提条件

- **min_users**: 6パターン固定（5, 10, 50, 100, 500, 1000）
- **1日あたりのエントリー数**: 400件未満
- **データサイズ**: 400件 × 約0.5KB = 約200KB（JSON、圧縮前）
- **圧縮後**: 約60-80KB（Snappy圧縮）

### 採用戦略: 日別全件キャッシュ + アプリ層フィルタ

この規模であれば、**日付ごとに全エントリーをキャッシュし、アプリケーション層でフィルタリング**するのが最適です。

#### 理由

1. **キャッシュキーが最小限**: 日付のみ（1日1キー）
2. **キャッシュヒット率100%**: すべてのパラメータの組み合わせに対応
3. **無効化がシンプル**: 日付単位で削除するだけ
4. **データサイズが小さい**: 圧縮後60-80KBなら転送コストは許容範囲
5. **/entries/newと/entries/hotで共用可能**: 同じデータを並び替えるだけ

#### デメリットの評価

- ~~大量データ転送~~ → 400件は小規模、圧縮後60-80KBは問題なし
- ~~メモリ使用量大~~ → 1日1キーなので逆に少ない
- アプリ層フィルタ処理 → 400件のフィルタは数ミリ秒で完了

---

### エンドポイント別キャッシュ戦略（再設計）

#### 1. `/entries/new` と `/entries/hot`

**キャッシュキー**: `hateblog:entries:{date}:all`

- 両エンドポイントで**同じキャッシュを共用**
- キャッシュには日付のエントリー全件（400件未満）を保存
- アプリ層で以下の処理を実行:
  1. min_usersでフィルタリング
  2. ソート（new: posted_at DESC、hot: bookmark_count DESC, posted_at DESC）
  3. limit/offsetでページング

**TTL**: 5分

**実装例**:

```go
// キャッシュキー
cacheKey := fmt.Sprintf("hateblog:entries:%s:all", date)

// キャッシュから全件取得
var allEntries []Entry
if err := cache.Get(ctx, cacheKey, &allEntries); err != nil {
    // キャッシュミス時はDBから取得
    allEntries = fetchEntriesFromDB(date)
    cache.Set(ctx, cacheKey, allEntries, 5*time.Minute)
}

// アプリ層でフィルタリング
filtered := filterByMinUsers(allEntries, minUsers)

// ソート
if endpoint == "hot" {
    sort.Slice(filtered, func(i, j int) bool {
        if filtered[i].BookmarkCount != filtered[j].BookmarkCount {
            return filtered[i].BookmarkCount > filtered[j].BookmarkCount
        }
        return filtered[i].PostedAt.After(filtered[j].PostedAt)
    })
} // newはすでにposted_at DESCでソート済み

// ページング
return paginate(filtered, limit, offset)
```

**効果**:
- キャッシュキー数: 1日1個（パラメータ組み合わせ × ページ数分削減）
- キャッシュヒット率: 100%
- DB負荷軽減: 95%以上

---

#### 2. `/tags/entries/{tag}`

**キャッシュキー**: `hateblog:tags:{tag_name}:entries:{sort}:{min_users}:100:0`

- limitパラメータは最大100
- `limit=100` かつ `offset=0` のときのみ100件をキャッシュ
- それ以外はDB取得のみ（キャッシュしない）

**TTL**: 10分

**実装例**:

```go
cacheKey := fmt.Sprintf("hateblog:tags:%s:entries:%s:%d:100:0", tagName, sort, minUsers)

var entries []Entry
if limit == 100 && offset == 0 {
    if err := cache.Get(ctx, cacheKey, &entries); err != nil {
        entries = fetchEntriesByTagFromDB(tagName, sort, minUsers, 100, 0)
        cache.Set(ctx, cacheKey, entries, 10*time.Minute)
    }
} else {
    entries = fetchEntriesByTagFromDB(tagName, sort, minUsers, limit, offset)
}
return entries
```

**効果**:
- 人気タグのDB負荷を大幅削減
- `limit=100&offset=0` のみキャッシュ

---

#### 3. `/rankings/yearly`

**キャッシュキー**: `hateblog:rankings:yearly:{year}:{min_users}`

- limitパラメータは最大100
- `limit=100` かつ `offset=0` のときのみ100件をキャッシュ
- それ以外はDB取得のみ（キャッシュしない）

**TTL**:
- 過去年: 7日間（ほぼ不変）
- 当年: 1時間

```go
cacheKey := fmt.Sprintf("hateblog:rankings:yearly:%d:%d", year, minUsers)

var entries []Entry
if limit == 100 && offset == 0 {
    if err := cache.Get(ctx, cacheKey, &entries); err != nil {
        entries = fetchYearlyRankingFromDB(year, 100)
        ttl := 7 * 24 * time.Hour
        if year == time.Now().Year() {
            ttl = 1 * time.Hour
        }
        cache.Set(ctx, cacheKey, entries, ttl)
    }
} else {
    entries = fetchYearlyRankingFromDBWithOffset(year, limit, offset)
}
return entries
```

---

#### 4. `/rankings/monthly`

**キャッシュキー**: `hateblog:rankings:monthly:{year}:{month}:{min_users}`

- limitパラメータは最大100
- `limit=100` かつ `offset=0` のときのみ100件をキャッシュ
- それ以外はDB取得のみ（キャッシュしない）

**TTL**:
- 過去月: 24時間
- 当月: 1時間

```go
cacheKey := fmt.Sprintf("hateblog:rankings:monthly:%d:%d:%d", year, month, minUsers)

var entries []Entry
if limit == 100 && offset == 0 {
    if err := cache.Get(ctx, cacheKey, &entries); err != nil {
        entries = fetchMonthlyRankingFromDB(year, month, 100)
        ttl := 24 * time.Hour
        if year == now.Year() && month == int(now.Month()) {
            ttl = 1 * time.Hour
        }
        cache.Set(ctx, cacheKey, entries, ttl)
    }
} else {
    entries = fetchMonthlyRankingFromDBWithOffset(year, month, limit, offset)
}
return entries
```

---

#### 5. `/rankings/weekly`

**キャッシュキー**: `hateblog:rankings:weekly:{year}:{week}:{min_users}`

**TTL**:
- 過去週: 24時間
- 当週: 30分

limitパラメータは最大100で、`limit=100` かつ `offset=0` のときのみ100件をキャッシュ。それ以外はDB取得のみ。

---

#### 6. `/archive`

**キャッシュキー**: `hateblog:archive:{year}:{month}`

- min_usersが6パターンあるが、集計結果は小さい（月別・日別の件数のみ）
- min_usersごとに別キーでキャッシュ

```go
cacheKey := fmt.Sprintf("hateblog:archive:%d:%s:%d", year, monthStr, minUsers)
```

**TTL**: 24時間

---

#### 7. `/tags`

**キャッシュキー**: `hateblog:tags:list:all`

- 全タグを1つのキーにキャッシュ
- アプリ層でlimit/offsetを適用

**TTL**: 1時間

```go
cacheKey := "hateblog:tags:list:all"

var allTags []Tag
if err := cache.Get(ctx, cacheKey, &allTags); err != nil {
    allTags = fetchAllTagsFromDB() // エントリー数でソート済み
    cache.Set(ctx, cacheKey, allTags, 1*time.Hour)
}

return paginate(allTags, limit, offset)
```

---

#### 8. `/search`

**キャッシュキー**: `hateblog:search:{query_hash}:{sort}:{min_users}:100:0`

- limitパラメータは最大100
- `limit=100` かつ `offset=0` のときのみ100件をキャッシュ
- それ以外はDB取得のみ（キャッシュしない）

**TTL**: 15分

```go
queryHash := sha256.Sum256([]byte(query))
cacheKey := fmt.Sprintf("hateblog:search:%x:%s:%d:100:0", queryHash, sort, minUsers)

var results []Entry
if limit == 100 && offset == 0 {
    if err := cache.Get(ctx, cacheKey, &results); err != nil {
        results = searchFromDB(query, sort, minUsers, 100, 0)
        cache.Set(ctx, cacheKey, results, 15*time.Minute)
    }
} else {
    results = searchFromDB(query, sort, minUsers, limit, offset)
}
return results
```

---

### キャッシュキー一覧（再設計版）

| エンドポイント | キャッシュキー | キー数 |
|--------------|--------------|-------|
| `/entries/new` | `hateblog:entries:{date}:all` | 日付数 |
| `/entries/hot` | `hateblog:entries:{date}:all` | **同上（共用）** |
| `/tags/entries/{tag}` | `hateblog:tags:{tag}:entries:{sort}:{min_users}:100:0` | タグ数 × sort × min_users |
| `/rankings/yearly` | `hateblog:rankings:yearly:{year}:{min_users}` | 年数 × min_users |
| `/rankings/monthly` | `hateblog:rankings:monthly:{year}:{month}:{min_users}` | 年月数 × min_users |
| `/rankings/weekly` | `hateblog:rankings:weekly:{year}:{week}:{min_users}` | 年週数 × min_users |
| `/archive` | `hateblog:archive:{year}:{month}:{min_users}` | 年月 × 6 |
| `/tags` | `hateblog:tags:list:all` | 1 |
| `/search` | `hateblog:search:{query_hash}:{sort}:{min_users}:100:0` | クエリ数 × sort × min_users |

**総キャッシュキー数の削減効果**:
- 旧設計: 数千〜数万キー（パラメータ × ページ数）
- 新設計: 数百キー（日付・タグ・クエリ単位）

---

### 共通ユーティリティ関数

```go
// min_usersでフィルタリング
func filterByMinUsers(entries []Entry, minUsers int) []Entry {
    filtered := make([]Entry, 0, len(entries))
    for _, entry := range entries {
        if entry.BookmarkCount >= minUsers {
            filtered = append(filtered, entry)
        }
    }
    return filtered
}

// ページング
func paginate(entries []Entry, limit, offset int) []Entry {
    if offset >= len(entries) {
        return []Entry{}
    }
    end := offset + limit
    if end > len(entries) {
        end = len(entries)
    }
    return entries[offset:end]
}
```

---

## キャッシュデータの圧縮戦略

### Redis自体の圧縮機能

**結論: Redisには透過的な圧縮機能はない**

- Redisは内部的にデータを圧縮せず、そのまま保存します
- 圧縮が必要な場合は、**アプリケーション層で実装**する必要があります

---

### アプリケーション層での圧縮実装

#### 圧縮すべきかの判断基準

| データサイズ | 圧縮推奨 | 理由 |
|------------|---------|------|
| < 1KB | **不要** | 圧縮オーバーヘッドの方が大きい |
| 1KB - 10KB | **ケースバイケース** | データ特性による（テキスト多めなら有効） |
| > 10KB | **推奨** | メモリ削減とネットワーク転送量削減の効果大 |

#### 圧縮アルゴリズムの比較

| アルゴリズム | 圧縮率 | 速度 | Go実装 | 推奨用途 |
|------------|-------|-----|--------|---------|
| **Gzip** | 高 | 中 | `compress/gzip` | 汎用、互換性重視 |
| **Snappy** | 中 | 非常に高速 | `github.com/golang/snappy` | **推奨**: 速度重視 |
| **LZ4** | 中 | 非常に高速 | `github.com/pierrec/lz4` | Snappyの代替 |
| **Zstd** | 非常に高 | 高速 | `github.com/klauspost/compress/zstd` | 圧縮率重視 |

**推奨: Snappy**
- 圧縮・解凍が非常に高速（CPU負荷が低い）
- Googleが開発、広く使われている
- Go標準ライブラリに近い安定性

---

### 実装例: Snappy圧縮

```go
package cache

import (
    "context"
    "encoding/json"
    "fmt"
    "github.com/golang/snappy"
    "github.com/redis/go-redis/v9"
)

type CompressedCache struct {
    client            *redis.Client
    compressionThreshold int // この閾値を超えたら圧縮
}

// Set: データをJSON化 → 圧縮 → Redis保存
func (c *CompressedCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
    // JSON シリアライズ
    jsonData, err := json.Marshal(value)
    if err != nil {
        return fmt.Errorf("json marshal: %w", err)
    }

    var data []byte
    compressed := false

    // 閾値を超えたら圧縮
    if len(jsonData) > c.compressionThreshold {
        data = snappy.Encode(nil, jsonData)
        compressed = true
    } else {
        data = jsonData
    }

    // 圧縮フラグをキーに含める（または別フィールドで管理）
    if compressed {
        key = key + ":z" // :z サフィックスで圧縮済みを示す
    }

    return c.client.Set(ctx, key, data, ttl).Err()
}

// Get: Redis取得 → 解凍 → JSONデコード
func (c *CompressedCache) Get(ctx context.Context, key string, dest interface{}) error {
    // 圧縮版のキーで取得
    compressedKey := key + ":z"
    data, err := c.client.Get(ctx, compressedKey).Bytes()

    if err != nil {
        return err
    }

    // 解凍
    jsonData, err := snappy.Decode(nil, data)
    if err != nil {
        return fmt.Errorf("snappy decode: %w", err)
    }

    // JSON デシリアライズ
    return json.Unmarshal(jsonData, dest)
}
```

---

### エンドポイント別の圧縮推奨

| エンドポイント | データサイズ想定 | 圧縮推奨 | 閾値 |
|--------------|---------------|---------|------|
| `/entries/new` | 5-20KB (25件) | **推奨** | 5KB |
| `/entries/hot` | 5-20KB (25件) | **推奨** | 5KB |
| `/rankings/*` | 10-50KB (100件) | **強く推奨** | 5KB |
| `/search` | 5-20KB | **推奨** | 5KB |
| `/tags/entries/{tag}` | 5-20KB | **推奨** | 5KB |
| `/archive` | 1-5KB | ケースバイケース | 10KB |
| `/tags` | 10-50KB | **推奨** | 5KB |

**推奨閾値: 5KB**
- 5KB以上のデータは圧縮することでメモリとネットワーク帯域を節約
- 5KB未満は圧縮オーバーヘッドを避けるため非圧縮

---

### 圧縮効果の試算

#### エントリーリストの例（25件、JSONフォーマット）

```
非圧縮: 15KB
Snappy圧縮後: 4-6KB (圧縮率 60-70%)
```

#### 効果

**メモリ削減**:
- 1000個のキャッシュ × 10KB削減 = **10MB削減**
- 大規模運用では数百MB〜GB単位の削減

**ネットワーク転送量削減**:
- APIレスポンスが高速化（特にRedisとアプリが別サーバーの場合）
- 10KB → 5KBで転送時間が半減

**CPU負荷**:
- Snappyは非常に軽量（圧縮: 数マイクロ秒、解凍: 数マイクロ秒）
- DB負荷削減効果の方が圧倒的に大きい

---

### 実装チェックリスト（圧縮対応）

- [ ] Snappyライブラリのインポート (`github.com/golang/snappy`)
- [ ] 圧縮閾値の設定（推奨: 5KB）
- [ ] キャッシュSet時の自動圧縮処理
- [ ] キャッシュGet時の自動解凍処理
- [ ] 圧縮率のメトリクス収集

---

## 今後の最適化案

- **CDN統合**: 静的レスポンスのエッジキャッシュ
- **キャッシュ階層化**: L1 (インメモリ) + L2 (Redis)
- **プリフェッチ**: ユーザー行動予測に基づくキャッシュ
- **A/Bテスト**: TTL値の最適化実験
- **動的パラメータ最適化**: アクセスログに基づくキャッシュ対象の自動調整
- **圧縮アルゴリズムの比較検証**: Snappy vs Zstd のベンチマーク
