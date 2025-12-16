#!/usr/bin/env bash
# このスクリプトをサーバー上で実行してください
# SSH: ssh grik@162.43.74.38
# 実行: bash /tmp/manual-setup-commands.sh

set -euo pipefail

echo "==== Shiboroom サーバーセットアップ ===="
echo ""

# 1. systemdサービスファイルを配置
echo "[1] systemdサービスファイルを配置..."
sudo mv /tmp/shiboroom-backend.service /etc/systemd/system/
sudo mv /tmp/shiboroom-frontend.service /etc/systemd/system/

# 2. Nginx設定ファイルを配置
echo "[2] Nginx設定ファイルを配置..."
sudo mv /tmp/nginx-shiboroom.conf /etc/nginx/sites-available/shiboroom.conf
sudo ln -sf /etc/nginx/sites-available/shiboroom.conf /etc/nginx/sites-enabled/

# 3. Nginx設定テスト
echo "[3] Nginx設定をテスト..."
sudo nginx -t

# 4. systemdサービスを有効化
echo "[4] systemdサービスを有効化..."
sudo systemctl daemon-reload
sudo systemctl enable shiboroom-backend
sudo systemctl enable shiboroom-frontend

# 5. sudoers設定（サービス再起動権限）
echo "[5] sudoers設定を追加..."
echo "grik ALL=(ALL) NOPASSWD: /bin/systemctl restart shiboroom-backend, /bin/systemctl restart shiboroom-frontend, /bin/systemctl status shiboroom-backend, /bin/systemctl status shiboroom-frontend" | sudo tee /etc/sudoers.d/shiboroom-services
sudo chmod 0440 /etc/sudoers.d/shiboroom-services

echo ""
echo "✅ 基本セットアップ完了"
echo ""
echo "次のステップ:"
echo "1. MySQLデータベースをセットアップ（以下のコマンドを実行）:"
echo "   mysql -u root -p"
echo "   CREATE DATABASE shiboroom CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
echo "   CREATE USER 'shiboroom_user'@'localhost' IDENTIFIED BY 'YOUR_PASSWORD';"
echo "   GRANT ALL PRIVILEGES ON shiboroom.* TO 'shiboroom_user'@'localhost';"
echo "   FLUSH PRIVILEGES;"
echo "   EXIT;"
echo ""
echo "2. /var/www/shiboroom/config/scraper_config.yaml を編集してDBパスワードを設定"
echo ""
echo "3. Let's Encrypt SSL証明書を取得:"
echo "   sudo certbot --nginx -d shiboroom.com -d www.shiboroom.com"
echo ""
echo "4. サービスを起動:"
echo "   sudo systemctl start shiboroom-backend"
echo "   sudo systemctl start shiboroom-frontend"
echo "   sudo systemctl reload nginx"
echo ""
