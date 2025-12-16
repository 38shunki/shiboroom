#!/usr/bin/env bash
# ステップ3: Let's Encrypt SSL証明書取得とNginx HTTPS設定
# サーバー上で実行: bash /tmp/setup-step3-ssl.sh

set -euo pipefail

echo "==== [ステップ3] SSL証明書取得とHTTPS設定 ===="
echo ""

# 1. certbotがインストールされているか確認
if ! command -v certbot &> /dev/null; then
    echo "[1/4] certbotをインストール..."
    sudo apt update
    sudo apt install -y certbot python3-certbot-nginx
else
    echo "[1/4] certbot は既にインストール済み"
fi

# 2. Let's Encrypt証明書を取得
echo "[2/4] Let's Encrypt証明書を取得..."
echo "メールアドレスの入力と利用規約への同意が必要です"
echo "Nginxを一時停止して証明書を取得します..."
echo ""
sudo systemctl stop nginx
sudo certbot certonly --standalone -d shiboroom.com -d www.shiboroom.com
sudo systemctl start nginx

# 3. HTTPS対応のNginx設定に更新
echo "[3/4] Nginx設定をHTTPS対応に更新..."
sudo tee /etc/nginx/sites-available/shiboroom.conf > /dev/null <<'EOF'
# Nginx設定: shiboroom.com

# HTTPからHTTPSへリダイレクト
server {
    listen 80;
    listen [::]:80;
    server_name shiboroom.com www.shiboroom.com;

    # Let's Encrypt証明書取得用
    location /.well-known/acme-challenge/ {
        root /var/www/certbot;
    }

    # その他すべてHTTPSへリダイレクト
    location / {
        return 301 https://$server_name$request_uri;
    }
}

# HTTPS設定
server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name shiboroom.com www.shiboroom.com;

    # SSL証明書（Let's Encrypt）
    ssl_certificate /etc/letsencrypt/live/shiboroom.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/shiboroom.com/privkey.pem;

    # SSL設定
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_prefer_server_ciphers on;
    ssl_ciphers 'ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384';
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;

    # セキュリティヘッダー
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;

    # ログ設定
    access_log /var/log/nginx/shiboroom-access.log;
    error_log /var/log/nginx/shiboroom-error.log;

    # クライアントアップロードサイズ制限
    client_max_body_size 10M;

    # バックエンドAPI（/api/* へのリクエスト）
    location /api/ {
        proxy_pass http://127.0.0.1:8085;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;

        # タイムアウト設定
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    # フロントエンド（Next.js）
    location / {
        proxy_pass http://127.0.0.1:5177;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;

        # タイムアウト設定
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    # Next.js静的ファイル（_next/*）
    location /_next/static/ {
        proxy_pass http://127.0.0.1:5177;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_cache_valid 200 1y;
        add_header Cache-Control "public, immutable";
    }

    # favicon, robots.txt等
    location ~* \.(ico|css|js|gif|jpe?g|png|svg|woff|woff2|ttf|eot)$ {
        proxy_pass http://127.0.0.1:5177;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
}
EOF

# 4. Nginx設定テストとリロード
echo "[4/4] Nginx設定テストとリロード..."
sudo nginx -t
sudo systemctl reload nginx

# 5. 証明書の自動更新設定
echo "[追加] 証明書の自動更新を設定..."
sudo tee /etc/cron.d/certbot-renew > /dev/null <<'EOF'
0 0 1 * * root certbot renew --deploy-hook "systemctl reload nginx" >> /var/log/letsencrypt/renew.log 2>&1
EOF

echo ""
echo "✅ SSL証明書取得とHTTPS設定完了！"
echo ""
echo "次のステップ:"
echo "  bash /tmp/setup-step4-start.sh"
echo ""
