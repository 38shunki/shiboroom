#!/usr/bin/env bash
# ステップ2: MySQLデータベースセットアップ
# サーバー上で実行: bash /tmp/setup-step2-database.sh
# 注意: MySQLのrootパスワードが必要です

set -euo pipefail

echo "==== [ステップ2] MySQLデータベースセットアップ ===="
echo ""

# データベースパスワードを入力してもらう
read -sp "新しいshiboroom_userのパスワードを入力してください: " DB_PASSWORD
echo ""

# MySQLに接続してデータベース作成
echo "[1/2] MySQLデータベースとユーザーを作成..."
sudo mysql -u root <<EOF
CREATE DATABASE IF NOT EXISTS shiboroom CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER IF NOT EXISTS 'shiboroom_user'@'localhost' IDENTIFIED BY '$DB_PASSWORD';
GRANT ALL PRIVILEGES ON shiboroom.* TO 'shiboroom_user'@'localhost';
FLUSH PRIVILEGES;
EOF

echo "[2/2] 設定ファイルにパスワードを設定..."
# 設定ファイルのパスワードを更新
sudo sed -i "s/password: \".*\"/password: \"$DB_PASSWORD\"/" /var/www/shiboroom/config/scraper_config.yaml

echo ""
echo "✅ データベースセットアップ完了！"
echo ""
echo "データベース情報:"
echo "  データベース名: shiboroom"
echo "  ユーザー名: shiboroom_user"
echo "  パスワード: [設定済み]"
echo ""
echo "次のステップ:"
echo "  bash /tmp/setup-step3-ssl.sh"
echo ""
