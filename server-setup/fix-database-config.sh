#!/usr/bin/env bash
# データベース設定を修正
# サーバー上で実行: bash /tmp/fix-database-config.sh

set -euo pipefail

echo "==== データベース設定の修正 ===="
echo ""

CONFIG_FILE="/var/www/shiboroom/config/scraper_config.yaml"

# 現在のパスワードを保持しながらユーザー名とデータベース名を修正
echo "[1/2] 設定ファイルを修正..."

# MySQLユーザー名を修正
sed -i 's/user: "realestate_user"/user: "shiboroom_user"/g' "$CONFIG_FILE"

# データベース名を修正
sed -i 's/database: "realestate_db"/database: "shiboroom"/g' "$CONFIG_FILE"

echo "[2/2] バックエンドを再起動..."
sudo systemctl restart shiboroom-backend
sleep 3

echo ""
echo "✅ 設定修正完了"
echo ""
echo "ステータス確認:"
journalctl -u shiboroom-backend -n 10 --no-pager
echo ""
