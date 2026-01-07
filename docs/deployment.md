# Deployment Guide

## Overview

開発・本番ともに Docker Compose を使用します。

- **開発**: `compose.yaml` + `compose.override.yaml` (自動読み込み)
- **本番**: `compose.yaml` のみ

## Development Environment

### Setup

```bash
# リポジトリクローン
git clone <repository-url>
cd hateblog-backend

# 環境変数ファイルをコピー
cp .env.example .env

# コンテナ起動（compose.override.yaml が自動で読み込まれる）
docker compose up -d

# appコンテナに入る
docker compose exec app sh

# 中でアプリケーション実行
go run cmd/app/main.go
```

または Dev Container を使用（推奨）:
```
VSCode コマンドパレット → Dev Containers: Reopen in Container
```

### 停止

```bash
docker compose down
```

## Production Deployment (VPS)

### Prerequisites

```bash
# システムアップデート
sudo apt update && sudo apt upgrade -y

# Docker と Docker Compose のインストール
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo apt install docker-compose-plugin -y
sudo usermod -aG docker $USER
newgrp docker

# 動作確認
docker --version
docker compose version
```

### 1. プロジェクトファイルの配置

```bash
# プロジェクトディレクトリ作成
sudo mkdir -p /opt/hateblog
sudo chown $USER:$USER /opt/hateblog
cd /opt/hateblog

# 必要なファイルのみをコピー（git clone または rsync）
# compose.yaml, Dockerfile, .dockerignore, cmd/, internal/, go.mod, go.sum など

# 重要: compose.override.yaml は本番に配置しない
```

### 2. 環境変数の設定

`.env` ファイルを作成：

```bash
cat > /opt/hateblog/.env <<'EOF'
# PostgreSQL
POSTGRES_PASSWORD=<strong-random-password>

# Application
APP_VERSION=1.0.0
POSTGRES_SSLMODE=require
APP_LOG_LEVEL=info

# Optional: カスタム設定
POSTGRES_MAX_CONNS=50
POSTGRES_MIN_CONNS=10
EOF

# パーミッション設定
chmod 600 /opt/hateblog/.env
```

**重要**: パスワードは必ず強力なランダム文字列に変更してください。

### 3. コンテナ起動

```bash
cd /opt/hateblog

# イメージビルドと起動
docker compose up -d --build

# ログ確認
docker compose logs -f

# ヘルスチェック（アプリ実装後）
curl http://127.0.0.1:8080/health
```

### 4. Nginx リバースプロキシ

```bash
sudo apt install nginx certbot python3-certbot-nginx -y
```

`/etc/nginx/sites-available/hateblog`:
```nginx
server {
    listen 80;
    server_name example.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name example.com;

    ssl_certificate /etc/letsencrypt/live/example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/example.com/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # タイムアウト設定
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }
}
```

```bash
sudo ln -s /etc/nginx/sites-available/hateblog /etc/nginx/sites-enabled/
sudo nginx -t
sudo certbot --nginx -d example.com
sudo systemctl reload nginx
```

## Monitoring

### ログ確認

```bash
cd /opt/hateblog

# アプリケーションログ
docker compose logs -f app

# 直近100行
docker compose logs --tail=100 app

# すべてのサービス
docker compose logs -f
```

### リソース監視

```bash
docker stats
docker compose ps
```

## Updates

### アプリケーション更新

```bash
cd /opt/hateblog

# 最新コードを取得（git pull または rsync）
git pull origin main

# 環境変数を更新（必要に応じて）
nano .env

# イメージ再ビルドと再起動
docker compose up -d --build

# ヘルスチェック
curl http://127.0.0.1:8080/health

# 古いイメージの削除
docker image prune -f
```

### ロールバック

```bash
# 前のバージョンにチェックアウト
git checkout <previous-commit-or-tag>

# 再起動
docker compose up -d --build
```

## Database Migration

マイグレーション実行（アプリ実装後）：

```bash
# マイグレーション適用
docker compose exec app /app migrate up

# ロールバック
docker compose exec app /app migrate down 1
```

## Batch Jobs（Cron運用）

フィード取得やブックマーク件数の定期更新は、ホスト側の cron で実行します。

### 前提条件

- Docker Compose が起動している状態
- `.env` ファイルが `/opt/hateblog/.env` に存在
- `docker compose` コマンドが実行可能

### セットアップ

ホスト側に crontab エントリを追加します：

```bash
# crontab -e で編集
crontab -e
```

以下を追加：

```crontab
# Hateblog batch jobs
# 環境はホストのシェル環境を使用
SHELL=/bin/bash
PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
HATEBLOG_DIR=/opt/hateblog

# Fetcher: 15分ごとに新規エントリーを取得
*/15 * * * * cd $HATEBLOG_DIR && docker compose exec -T app /fetcher >> /var/log/hateblog/fetcher.log 2>&1

# Updater - High priority: 5分ごとに高優先度エントリーの件数を更新
*/5 * * * * cd $HATEBLOG_DIR && docker compose exec -T app /updater --tier high >> /var/log/hateblog/updater-high.log 2>&1

# Updater - Medium priority: 30分ごと
*/30 * * * * cd $HATEBLOG_DIR && docker compose exec -T app /updater --tier medium >> /var/log/hateblog/updater-medium.log 2>&1

# Updater - Low priority: 毎時0分
0 * * * * cd $HATEBLOG_DIR && docker compose exec -T app /updater --tier low >> /var/log/hateblog/updater-low.log 2>&1

# Updater - Round: 毎日02:00 に全エントリーを循環更新
0 2 * * * cd $HATEBLOG_DIR && docker compose exec -T app /updater --tier round >> /var/log/hateblog/updater-round.log 2>&1
```

### ログディレクトリの作成

```bash
sudo mkdir -p /var/log/hateblog
sudo chown $(whoami):$(whoami) /var/log/hateblog
chmod 755 /var/log/hateblog
```

### ログローテーション

`/etc/logrotate.d/hateblog` を作成：

```bash
sudo tee /etc/logrotate.d/hateblog <<'EOF'
/var/log/hateblog/*.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
    create 0644 root root
}
EOF
```

### 同時実行の制御

各バッチはデータベース advisory lock を使用して同時実行を防止します（複数サーバー対応）。
ロック取得に失敗した場合、ジョブは自動的にスキップされます。

**ロック動作：**
- ロック取得成功 → ジョブ実行、終了コード 0
- ロック取得失敗 → スキップ、終了コード 0、ログに "lock not acquired; skip" と出力

### cron 実行確認

```bash
# cron ログを確認
grep CRON /var/log/syslog | tail -20

# または journalctl で確認
sudo journalctl -u cron --tail=20

# ジョブ実行ログ
tail -f /var/log/hateblog/fetcher.log
```

### フラグ設定

各ジョブの動作はフラグで制御可能：

**fetcher:**
- `--lock <name>` : advisory lock 名（デフォルト: fetcher）
- `--max-entries <n>` : 1回の実行で処理する最大エントリー数（デフォルト: 300）
- `--no-tags` : タグ抽出を無効化
- `--tag-top <n>` : 1エントリーあたりのタグ上限数（デフォルト: 5）
- `--deadline <duration>` : 実行タイムアウト（デフォルト: 5m）

**updater:**
- `--tier <tier>` : 更新優先度 (high|medium|low|round) - 必須
- `--lock <name>` : advisory lock 名（デフォルト: updater-<tier>）
- `--limit <n>` : 1回の実行で更新する最大エントリー数（デフォルト: 50）
- `--deadline <duration>` : 実行タイムアウト（デフォルト: 3m）

例：

```bash
# fetcher の タグ抽出を無効化、タイムアウト 10分
docker compose exec -T app /fetcher --no-tags --deadline 10m

# updater でタイムアウトを短縮
docker compose exec -T app /updater --tier high --deadline 2m
```

### トラブルシューティング

**ジョブが実行されない**

```bash
# cron デーモンが動作しているか確認
sudo systemctl status cron

# または
sudo systemctl restart cron

# ジョブログで lock 関連のエラーを確認
grep "lock" /var/log/hateblog/*.log
```

**外部API タイムアウト**

- `HATENA_API_TIMEOUT` を調整（デフォルト: 10s）
- `YAHOO_API_KEY` が設定されている場合、キー抽出がタイムアウト→ログに記録→再実行で自動回復

**ジョブ実行ログが見つからない**

- crontab で SHELL 環境変数を明示（bash を推奨）
- PATH を明示的に設定
- ディレクトリ変更 (`cd $HATEBLOG_DIR`) を明示



- [ ] 強力なパスワード設定（`.env` ファイル）
- [ ] `.env` ファイルのパーミッション（600）
- [ ] UFW ファイアウォール有効化（SSH、80、443 のみ許可）
- [ ] SSH 鍵認証のみ、パスワード認証無効化
- [ ] PostgreSQL/Redis は localhost バインド（127.0.0.1）
- [ ] HTTPS 強制リダイレクト（Nginx）
- [ ] fail2ban 導入
- [ ] 自動セキュリティアップデート（unattended-upgrades）
- [ ] 本番環境に `compose.override.yaml` を配置しない

```bash
# UFW 設定
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow ssh
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable

# SSH 強化（/etc/ssh/sshd_config）
PasswordAuthentication no
PermitRootLogin no
```

## Troubleshooting

### コンテナが起動しない

```bash
# ログ確認
docker compose logs

# 個別サービスのログ
docker compose logs postgres
docker compose logs app

# 再起動
docker compose restart app
```

### データベース接続エラー

```bash
# PostgreSQL ヘルスチェック
docker compose exec postgres pg_isready -U hateblog

# 接続テスト
docker compose exec postgres psql -U hateblog -d hateblog -c "SELECT 1;"
```

### ディスク容量不足

```bash
# 未使用リソースの削除
docker system prune -a --volumes -f

# ログのクリーンアップ
sudo journalctl --vacuum-time=7d
```

### compose.override.yaml が読み込まれてしまう

本番環境では `compose.override.yaml` を配置しないでください。誤って配置した場合：

```bash
rm compose.override.yaml
docker compose up -d --build
```

## File Structure

```
/opt/hateblog/
├── compose.yaml           # 本番設定（compose.override.yamlなし）
├── Dockerfile
├── .env                   # 本番用環境変数
├── cmd/
├── internal/
├── go.mod
└── go.sum
```

## References

- [Docker Compose Documentation](https://docs.docker.com/compose/)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/18/)
- [Redis Documentation](https://redis.io/docs/)
- [Nginx Documentation](https://nginx.org/en/docs/)
