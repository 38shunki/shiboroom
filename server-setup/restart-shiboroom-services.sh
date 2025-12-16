#!/usr/bin/env bash
# サーバー上で実行: /var/www/shiboroom/restart-shiboroom-services.sh

set -euo pipefail

echo "==== shiboroom サービス再起動 ===="

# バックエンド再起動
echo "→ shiboroom-backend を再起動..."
sudo systemctl restart shiboroom-backend
sleep 2
sudo systemctl status shiboroom-backend --no-pager || true

# フロントエンド再起動
echo "→ shiboroom-frontend を再起動..."
sudo systemctl restart shiboroom-frontend
sleep 2
sudo systemctl status shiboroom-frontend --no-pager || true

echo ""
echo "✅ shiboroom サービス再起動完了"
echo ""
echo "ステータス確認:"
echo "  sudo systemctl status shiboroom-backend"
echo "  sudo systemctl status shiboroom-frontend"
echo ""
echo "ログ確認:"
echo "  sudo journalctl -u shiboroom-backend -f"
echo "  sudo journalctl -u shiboroom-frontend -f"
