#!/usr/bin/env bash
# ステップ4: サービス起動と動作確認
# サーバー上で実行: bash /tmp/setup-step4-start.sh

set -euo pipefail

echo "==== [ステップ4] サービス起動と動作確認 ===="
echo ""

# 1. バックエンドサービス起動
echo "[1/4] バックエンドサービスを起動..."
sudo systemctl start shiboroom-backend
sleep 3
sudo systemctl status shiboroom-backend --no-pager || true

# 2. フロントエンドサービス起動
echo ""
echo "[2/4] フロントエンドサービスを起動..."
sudo systemctl start shiboroom-frontend
sleep 3
sudo systemctl status shiboroom-frontend --no-pager || true

# 3. Nginxステータス確認
echo ""
echo "[3/4] Nginxステータス確認..."
sudo systemctl status nginx --no-pager || true

# 4. ポート確認
echo ""
echo "[4/4] ポート確認..."
echo "バックエンド (8085):"
sudo lsof -i :8085 || echo "  ポート8085は使用されていません"
echo ""
echo "フロントエンド (5177):"
sudo lsof -i :5177 || echo "  ポート5177は使用されていません"

# 5. 簡易動作確認
echo ""
echo "[動作確認] APIヘルスチェック..."
sleep 2
curl -f http://localhost:8085/api/health 2>/dev/null && echo "" || echo "バックエンドAPIが応答しません"

echo ""
echo "=========================================="
echo "✅ セットアップ完了！"
echo "=========================================="
echo ""
echo "📊 サービス状態:"
echo "  バックエンド: systemctl status shiboroom-backend"
echo "  フロントエンド: systemctl status shiboroom-frontend"
echo ""
echo "📝 ログ確認:"
echo "  バックエンド: sudo journalctl -u shiboroom-backend -f"
echo "  フロントエンド: sudo journalctl -u shiboroom-frontend -f"
echo "  Nginx: sudo tail -f /var/log/nginx/shiboroom-error.log"
echo ""
echo "🌐 アクセスURL:"
echo "  https://shiboroom.com"
echo "  https://shiboroom.com/api/health"
echo ""
echo "🔄 サービス再起動:"
echo "  /var/www/shiboroom/restart-shiboroom-services.sh"
echo ""
