#!/usr/bin/env bash
# MySQLパスワードをリセット（sudoなしでroot接続）
# サーバー上で実行: sudo bash /tmp/reset-mysql-password-noroot.sh

set -euo pipefail

echo "==== MySQLパスワードのリセット ===="
echo ""

NEW_PASSWORD="Kihara0725$"

echo "[1/2] MySQLユーザーのパスワードを更新..."
mysql -u root <<EOF
ALTER USER 'shiboroom_user'@'localhost' IDENTIFIED BY '$NEW_PASSWORD';
FLUSH PRIVILEGES;
SELECT 'Password updated successfully' AS Status;
EOF

echo ""
echo "[2/2] バックエンドを再起動..."
systemctl restart shiboroom-backend
sleep 3

echo ""
echo "✅ パスワードリセット完了"
echo ""
echo "接続テスト:"
mysql -u shiboroom_user -p"$NEW_PASSWORD" -e "SELECT 'Connection successful!' AS Status;"
echo ""
echo "バックエンドステータス:"
journalctl -u shiboroom-backend -n 15 --no-pager
echo ""
