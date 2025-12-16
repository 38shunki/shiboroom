# Shiboroom.com サーバーセットアップ手順

## 前提条件

- サーバー: 162.43.74.38
- ユーザー: grik
- ドメイン: shiboroom.com
- 既存サービス: grik (別ポート稼働中)

## 1. サーバーへの初回セットアップ

### 1.1 ディレクトリ作成

```bash
ssh grik@162.43.74.38

# アプリケーションディレクトリ作成
sudo mkdir -p /var/www/shiboroom/{backend,frontend,config}
sudo chown -R grik:grik /var/www/shiboroom
```

### 1.2 データベースセットアップ（MySQL）

```bash
# MySQLに接続
mysql -u root -p

# データベースとユーザー作成
CREATE DATABASE shiboroom CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'shiboroom_user'@'localhost' IDENTIFIED BY 'STRONG_PASSWORD_HERE';
GRANT ALL PRIVILEGES ON shiboroom.* TO 'shiboroom_user'@'localhost';
FLUSH PRIVILEGES;
EXIT;
```

### 1.3 Meilisearchセットアップ

既存のMeilisearchが動作している場合はスキップ。新規の場合：

```bash
# Meilisearchインストール（Dockerを使用）
# または既存のMeilisearchインスタンスを使用
```

### 1.4 設定ファイルの配置

ローカルから設定ファイルをコピー：

```bash
# ローカルマシンから実行
scp backend/config/scraper_config.yaml grik@162.43.74.38:/var/www/shiboroom/config/

# サーバー上で設定ファイルを編集
ssh grik@162.43.74.38
nano /var/www/shiboroom/config/scraper_config.yaml
```

設定例（`scraper_config.yaml`）：
```yaml
database:
  mysql:
    host: "127.0.0.1"
    port: 3306
    user: "shiboroom_user"
    password: "STRONG_PASSWORD_HERE"
    database: "shiboroom"

search:
  meilisearch:
    host: "http://127.0.0.1:7700"
    api_key: "masterKey123"

scraper:
  request_delay_seconds: 2
  max_retries: 3
  max_requests_per_day: 5000
  daily_run_enabled: false
  daily_run_time: "02:00"

rate_limit:
  enabled: true
  requests_per_minute: 30
  requests_per_hour: 1800
```

## 2. systemdサービス設定

### 2.1 サービスファイルのコピー

```bash
# ローカルマシンから実行
scp server-setup/shiboroom-backend.service grik@162.43.74.38:/tmp/
scp server-setup/shiboroom-frontend.service grik@162.43.74.38:/tmp/
scp server-setup/restart-shiboroom-services.sh grik@162.43.74.38:/var/www/shiboroom/

# サーバー上で実行
ssh grik@162.43.74.38

# systemdサービスファイルを配置
sudo mv /tmp/shiboroom-backend.service /etc/systemd/system/
sudo mv /tmp/shiboroom-frontend.service /etc/systemd/system/

# 再起動スクリプトに実行権限を付与
chmod +x /var/www/shiboroom/restart-shiboroom-services.sh

# sudoersにサービス再起動権限を追加
sudo visudo
# 以下を追加:
# grik ALL=(ALL) NOPASSWD: /bin/systemctl restart shiboroom-backend, /bin/systemctl restart shiboroom-frontend, /bin/systemctl status shiboroom-backend, /bin/systemctl status shiboroom-frontend

# サービスを有効化
sudo systemctl daemon-reload
sudo systemctl enable shiboroom-backend
sudo systemctl enable shiboroom-frontend
```

## 3. Nginx設定

### 3.1 Nginx設定ファイルの配置

```bash
# ローカルマシンから実行
scp server-setup/nginx-shiboroom.conf grik@162.43.74.38:/tmp/

# サーバー上で実行
ssh grik@162.43.74.38
sudo mv /tmp/nginx-shiboroom.conf /etc/nginx/sites-available/shiboroom.conf
sudo ln -s /etc/nginx/sites-available/shiboroom.conf /etc/nginx/sites-enabled/

# Nginx設定テスト
sudo nginx -t
```

### 3.2 Let's EncryptでSSL証明書取得

```bash
# Certbotがインストールされていない場合
sudo apt update
sudo apt install certbot python3-certbot-nginx

# 証明書取得
sudo certbot --nginx -d shiboroom.com -d www.shiboroom.com

# 証明書の自動更新設定確認
sudo systemctl status certbot.timer

# テスト（ドライラン）
sudo certbot renew --dry-run
```

### 3.3 Nginx再起動

```bash
sudo systemctl reload nginx
```

## 4. 初回デプロイ

### 4.1 デプロイスクリプトに実行権限を付与

```bash
# ローカルマシンから実行
cd /Users/shu/Documents/dev/real-estate-portal
chmod +x deploy.sh
```

### 4.2 初回デプロイ実行

```bash
# ローカルマシンから実行
./deploy.sh
```

## 5. 動作確認

### 5.1 サービス状態確認

```bash
# サーバー上で実行
ssh grik@162.43.74.38

# バックエンド確認
sudo systemctl status shiboroom-backend
sudo journalctl -u shiboroom-backend -n 50

# フロントエンド確認
sudo systemctl status shiboroom-frontend
sudo journalctl -u shiboroom-frontend -n 50

# Nginx確認
sudo systemctl status nginx
sudo tail -f /var/log/nginx/shiboroom-error.log
```

### 5.2 ブラウザで確認

- https://shiboroom.com

### 5.3 API確認

```bash
curl https://shiboroom.com/api/properties
curl https://shiboroom.com/api/health
```

## 6. トラブルシューティング

### バックエンドが起動しない場合

```bash
# ログ確認
sudo journalctl -u shiboroom-backend -f

# 手動起動してエラー確認
cd /var/www/shiboroom/backend
CONFIG_PATH=/var/www/shiboroom/config/scraper_config.yaml ./shiboroom-api
```

### フロントエンドが起動しない場合

```bash
# ログ確認
sudo journalctl -u shiboroom-frontend -f

# Node.jsバージョン確認
node --version

# 手動起動してエラー確認
cd /var/www/shiboroom/frontend/.next/standalone
NODE_ENV=production PORT=5177 node server.js
```

### Nginxエラー

```bash
# 設定ファイル確認
sudo nginx -t

# エラーログ確認
sudo tail -f /var/log/nginx/shiboroom-error.log

# アクセスログ確認
sudo tail -f /var/log/nginx/shiboroom-access.log
```

### SSL証明書エラー

```bash
# 証明書確認
sudo certbot certificates

# 証明書の手動更新
sudo certbot renew
```

## 7. 定期メンテナンス

### ログローテーション

```bash
# ログサイズ確認
du -sh /var/log/nginx/shiboroom-*

# 古いログの削除（30日以上前）
find /var/log/nginx/shiboroom-* -mtime +30 -delete
```

### データベースバックアップ

```bash
# バックアップディレクトリ作成
mkdir -p /var/www/shiboroom/backups

# データベースバックアップ
mysqldump -u shiboroom_user -p shiboroom > /var/www/shiboroom/backups/shiboroom_$(date +%Y%m%d).sql

# 古いバックアップの削除（7日以上前）
find /var/www/shiboroom/backups/ -name "shiboroom_*.sql" -mtime +7 -delete
```

### 証明書更新

```bash
# 自動更新は設定済み、手動確認
sudo certbot renew
```

## 8. デプロイコマンド一覧

```bash
# 通常のデプロイ
./deploy.sh

# ログ確認（サーバー上）
sudo journalctl -u shiboroom-backend -f
sudo journalctl -u shiboroom-frontend -f

# サービス再起動（サーバー上）
sudo systemctl restart shiboroom-backend
sudo systemctl restart shiboroom-frontend

# Nginx再起動（サーバー上）
sudo systemctl reload nginx
```

## 9. ポート一覧

| サービス | ポート | 説明 |
|---------|--------|------|
| Frontend | 5177 | Next.js (内部) |
| Backend | 8085 | Go API (内部) |
| MySQL | 3306 | データベース (内部) |
| Meilisearch | 7700 | 検索エンジン (内部) |
| Nginx | 80, 443 | リバースプロキシ (外部) |

※ 外部からは443番ポート（HTTPS）のみアクセス可能

## 10. 環境変数一覧

### バックエンド
- `CONFIG_PATH`: /var/www/shiboroom/config/scraper_config.yaml
- `PORT`: 8085
- `ENV`: production

### フロントエンド
- `NODE_ENV`: production
- `PORT`: 5177
- `NEXT_PUBLIC_API_URL`: https://shiboroom.com
- `HOSTNAME`: 0.0.0.0
