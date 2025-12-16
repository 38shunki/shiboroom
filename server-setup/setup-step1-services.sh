#!/usr/bin/env bash
# ステップ1: systemdサービスとNginx基本設定
# サーバー上で実行: bash /tmp/setup-step1-services.sh

set -euo pipefail

echo "==== [ステップ1] systemdサービスとNginx基本設定 ===="
echo ""

# 1. systemdサービスファイルを配置
echo "[1/6] systemdサービスファイルを配置..."
sudo mv /tmp/shiboroom-backend.service /etc/systemd/system/
sudo mv /tmp/shiboroom-frontend.service /etc/systemd/system/

# 2. Nginx HTTP-only設定を作成（証明書取得前）
echo "[2/6] Nginx HTTP-only設定を作成..."
sudo tee /etc/nginx/sites-available/shiboroom.conf > /dev/null <<'EOF'
# Nginx設定: shiboroom.com (証明書取得用 - HTTP only)

server {
    listen 80;
    listen [::]:80;
    server_name shiboroom.com www.shiboroom.com;

    # Let's Encrypt証明書取得用
    location /.well-known/acme-challenge/ {
        root /var/www/certbot;
    }

    # ログ設定
    access_log /var/log/nginx/shiboroom-access.log;
    error_log /var/log/nginx/shiboroom-error.log;

    # バックエンドAPI
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
    }

    # フロントエンド
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
    }
}
EOF

# 3. シンボリックリンク作成
echo "[3/6] Nginx設定を有効化..."
sudo ln -sf /etc/nginx/sites-available/shiboroom.conf /etc/nginx/sites-enabled/

# 4. certbotディレクトリ作成
echo "[4/6] certbotディレクトリを作成..."
sudo mkdir -p /var/www/certbot

# 5. Nginx設定テスト
echo "[5/6] Nginx設定をテスト..."
sudo nginx -t

# 6. systemdサービスを有効化
echo "[6/6] systemdサービスを有効化..."
sudo systemctl daemon-reload
sudo systemctl enable shiboroom-backend
sudo systemctl enable shiboroom-frontend

# sudoers設定
echo "[追加] sudoers設定を追加..."
echo "grik ALL=(ALL) NOPASSWD: /bin/systemctl restart shiboroom-backend, /bin/systemctl restart shiboroom-frontend, /bin/systemctl status shiboroom-backend, /bin/systemctl status shiboroom-frontend, /bin/systemctl reload nginx" | sudo tee /etc/sudoers.d/shiboroom-services > /dev/null
sudo chmod 0440 /etc/sudoers.d/shiboroom-services

# Nginxリロード
echo "[最終] Nginxをリロード..."
sudo systemctl reload nginx

echo ""
echo "✅ ステップ1完了！"
echo ""
echo "次のステップ:"
echo "  bash /tmp/setup-step2-database.sh"
echo ""
