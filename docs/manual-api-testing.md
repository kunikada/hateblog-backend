# 手動APIテスト手順（シード付き）

## 前提
- `.env.example` を `.env` にコピー済み (`APP_API_KEY_REQUIRED=false` がデフォルト)
- Docker/Compose が動作すること
- 開発用は `compose.override.yaml` を併用すると `DB_SSLMODE=disable` になります

## 手順
1. サービス起動  
   ```bash
   docker compose -f compose.yaml -f compose.override.yaml up -d
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

## メモ
- データをリセットしたい場合は再度 `./scripts/seed_manual.sh` を実行してください。
- APIキー認証を有効にしている場合は `X-API-Key` ヘッダーを付けてください。
