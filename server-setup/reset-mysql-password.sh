#!/usr/bin/env bash
# MySQLパスワードをリセット
# サーバー上で実行: bash /tmp/reset-mysql-password.sh

set -euo pipefail

echo "==== MySQLパスワードのリセット ===="
echo ""

NEW_PASSWORD="Kihara0725$"

echo "[1/2] MySQLユーザーのパスワードを更新..."
sudo mysql -u root <<EOF
ALTER USER 'shiboroom_user'@'localhost' IDENTIFIED BY '$NEW_PASSWORD';
FLUSH PRIVILEGES;
EOF

echo "[2/2] バックエンドを再起動..."
sudo systemctl restart shiboroom-backend
sleep 3

echo ""
echo "✅ パスワードリセット完了"
echo ""
echo "ステータス確認:"
journalctl -u shiboroom-backend -n 15 --no-pager
echo ""
