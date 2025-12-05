# hateblog 技術要件（初稿）

UI/デザインは別途検討とし、ここでは技術的な前提のみを記載する。

## アーキテクチャ
- フロントエンド／バックエンド分離の SPA 構成
- API サーバーは一覧・詳細・ランキング・検索・タグ・アーカイブ・履歴などのデータ提供を担当
- レスポンシブ対応（スマホ・タブレット・PC）
- ログイン機能なし（履歴はブラウザローカルで保持する方針）
- API は REST で提供し、OpenAPI でスキーマ管理・型生成を行う

## パフォーマンス・運用
- 軽量でリソース消費を抑える設計（キャッシュ活用、必要最小限のデータ返却）
- 負荷分散／スケールを考慮した設計（ページネーション・クエリ最適化・キャッシュキー設計）
- 共有は Web Share API + SNS 公式シェア URL の組み合わせ。外部スクリプト（ウィジェット系）は極力使わず、使う場合も遅延ロードで最小化
- アナリティクス計測の実装方法は再検討（導入可否・手段を別途決定）

## データ要件（機能を支えるもの）
- エントリー：タイトル、リンク先 URL、投稿日、はてブ件数、抜粋、タグ一覧、favicon URL、共有用メタデータ
- フィルタ：はてブ件数の閾値パラメータ
- 日付軸：日付ごとの新着／人気、アーカイブ（年→月→日）、ランキング（年→月→週）
- タグ軸：タグ別一覧取得
- 検索：キーワード検索 API（対象フィールドは別途定義）
- 閲覧履歴：localStorage に保存＋表示用データ構造（サーバー側保存は行わない方針）
- クリック計測：記事クリックを記録する計測 API（エントリー ID をキー）

## API 例（たたき台）
- `GET /entries/new?date=YYYYMMDD&min_users=5&offset=...`
- `GET /entries/hot?date=YYYYMMDD&min_users=5&offset=...`
- `GET /archive?year=YYYY&month=MM&min_users=...`
- `GET /ranking/yearly?year=YYYY` / `GET /ranking/monthly?year=YYYY&month=MM` / `GET /ranking/weekly?year=YYYY&week=...`
- `GET /tags/{tag}?min_users=...&offset=...`
- `GET /search?q=...&offset=...`
- `POST /metrics/clicks`（エントリー ID・参照元などを送信）

※API パス／クエリ構造は後続の設計段階で調整可。*** End Patch
