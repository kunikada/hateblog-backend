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
DB_SSLMODE=require
LOG_LEVEL=info

# Optional: カスタム設定
DB_MAX_CONNS=50
DB_MIN_CONNS=10
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

## Backup

### PostgreSQL Backup

```bash
# バックアップスクリプト作成
cat > /opt/hateblog/backup.sh <<'EOF'
#!/bin/bash
BACKUP_DIR="/opt/hateblog/backups"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

mkdir -p $BACKUP_DIR

# バックアップ実行
docker compose exec -T postgres pg_dump -U hateblog hateblog | gzip > $BACKUP_DIR/backup_${TIMESTAMP}.sql.gz

# 古いバックアップ削除（30日以上前）
find $BACKUP_DIR -name "*.sql.gz" -mtime +30 -delete

echo "Backup completed: backup_${TIMESTAMP}.sql.gz"
EOF

chmod +x /opt/hateblog/backup.sh

# cron 設定（毎日午前3時）
(crontab -l 2>/dev/null; echo "0 3 * * * cd /opt/hateblog && ./backup.sh >> /var/log/hateblog-backup.log 2>&1") | crontab -
```

### リストア手順

```bash
# バックアップから復元
cd /opt/hateblog
gunzip -c backups/backup_20250101_030000.sql.gz | docker compose exec -T postgres psql -U hateblog hateblog
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

## Security Checklist

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
├── go.sum
├── backups/               # バックアップ保存先
└── backup.sh              # バックアップスクリプト
```

## References

- [Docker Compose Documentation](https://docs.docker.com/compose/)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/18/)
- [Redis Documentation](https://redis.io/docs/)
- [Nginx Documentation](https://nginx.org/en/docs/)
