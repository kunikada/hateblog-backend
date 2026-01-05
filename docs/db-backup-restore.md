# PostgreSQL バックアップ / 復元（pg_dump 日次フル）

## スコープ
- 対象: `compose.yaml` の `postgres` サービス（PostgreSQL 18）
- 方針: pg_dump による日次フルバックアップのみ。暗号化・高度な仕組みなし。
- 保存先: S3 バケット（リージョン/バケット名/プレフィックスは任意に設定）

## 事前準備（AWS 側）
1) S3 バケット作成（例: バケット名 `hateblog-backup`, リージョン `ap-northeast-1`）  
2) IAM ユーザー作成（プログラムアクセスのみ）。S3 バケットに限定した権限を付与（Put/Get/List）。例:
   ```json
   {
     "Version": "2012-10-17",
     "Statement": [
       { "Effect": "Allow", "Action": ["s3:PutObject","s3:GetObject","s3:ListBucket"], "Resource": ["arn:aws:s3:::<bucket>","arn:aws:s3:::<bucket>/*"] }
     ]
   }
   ```
3) AWS CLI を実行する環境でクレデンシャル設定（いずれか）
   - `aws configure --profile <profile-name>` を実行
   - もしくは環境変数 `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` / `AWS_DEFAULT_REGION` をセット

## バックアップ（スクリプト）
`scripts/backup_pg_dump_s3.sh` を実行するだけ。`.env` があれば `POSTGRES_PASSWORD` / `DB_NAME` / `DB_USER` を自動で読み込む。

必須: `S3_BUCKET`  
任意: `S3_PREFIX`（デフォルト `db-backups/`）、`AWS_PROFILE` もしくは `AWS_ACCESS_KEY_ID`/`AWS_SECRET_ACCESS_KEY` など

```bash
S3_BUCKET=<bucket> S3_PREFIX=db-backups/ AWS_PROFILE=<profile-name> \
  bash scripts/backup_pg_dump_s3.sh
```
- `docker compose exec` でコンテナ内の `pg_dump` を実行し、`gzip` → `aws s3 cp` まで一括で行う。
- 実行ログは標準出力に出るため、必要ならリダイレクトで保存する。

### 日次スケジュール例（cron）
例: サーバー上の `/opt/hateblog` で毎日 03:00 に実行し、ログを残す。
```
0 3 * * * cd /opt/hateblog && \
  S3_BUCKET=<bucket> S3_PREFIX=db-backups/ AWS_PROFILE=<profile-name> \
  bash scripts/backup_pg_dump_s3.sh >> /var/log/hateblog-backup.log 2>&1
```
- ログファイルのパーミッションに注意。
- バックアップ保持は S3 のライフサイクルルールで管理する（例: 30/90 日で削除）。

## 復元（スクリプト）
対象のバックアップを指定して実行する。必要ならアプリを停止してから実施。

```bash
S3_BUCKET=<bucket> S3_PREFIX=db-backups/ BACKUP_NAME=backup_YYYYMMDD_HHMMSS.sql.gz \
  AWS_PROFILE=<profile-name> bash scripts/restore_pg_dump_s3.sh
```

`S3_KEY` を指定する場合:
```bash
S3_BUCKET=<bucket> S3_KEY=<prefix>/backup_YYYYMMDD_HHMMSS.sql.gz \
  AWS_PROFILE=<profile-name> bash scripts/restore_pg_dump_s3.sh
```

- 復元後、最低限の確認を実施（例: `docker compose exec postgres psql -U hateblog -c "SELECT 1"`）。
- 復元は既存データを上書きするため、実行前に必ず意図を確認する。

## 運用メモ
- バックアップ取得・復元の作業ユーザーに S3 アクセスキーを渡しすぎない（必要最小限に限定）。
- 空き容量・権限エラーで失敗しやすいので、最初に手動実行で成功確認し、その後 cron に登録する。
