#!/usr/bin/env bash
# Meilisearchのマスターキーを修正
# サーバー上で実行: bash /tmp/fix-meilisearch-key.sh

set -euo pipefail

echo "==== Meilisearch マスターキー修正 ===="
echo ""

# 新しい安全なマスターキー（16バイト以上）
NEW_KEY="p5l05tZv4J5j3SXR2W1yGifbo4s94fe6Fv_YuPSrBfY"

echo "[1/4] Meilisearch systemdサービスファイルを更新..."
sudo tee /etc/systemd/system/meilisearch.service > /dev/null <<EOF
[Unit]
Description=Meilisearch
After=network.target

[Service]
Type=simple
User=grik
Group=grik
WorkingDirectory=/var/www/shiboroom
ExecStart=/usr/local/bin/meilisearch --http-addr 127.0.0.1:7700 --master-key $NEW_KEY --db-path /var/www/shiboroom/meilisearch-data

# 環境変数
Environment="MEILI_ENV=production"
Environment="MEILI_HTTP_ADDR=127.0.0.1:7700"
Environment="MEILI_MASTER_KEY=$NEW_KEY"

# ログ設定
StandardOutput=journal
StandardError=journal
SyslogIdentifier=meilisearch

# 再起動設定
Restart=always
RestartSec=5

# セキュリティ設定
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOF

echo "[2/4] scraper_config.yamlのMeilisearchキーを更新..."
sed -i "s|api_key:.*|api_key: \"$NEW_KEY\"|" /var/www/shiboroom/config/scraper_config.yaml

echo "[3/4] Meilisearchサービスを再起動..."
sudo systemctl daemon-reload
sudo systemctl restart meilisearch
sleep 3

# 起動確認
if sudo systemctl is-active --quiet meilisearch; then
    echo "✅ Meilisearch起動成功"
    sudo systemctl status meilisearch --no-pager || true
else
    echo "❌ Meilisearch起動失敗"
    sudo journalctl -u meilisearch -n 20 --no-pager
    exit 1
fi

echo ""
echo "[4/4] バックエンドを再起動..."
sudo systemctl restart shiboroom-backend
sleep 3

if sudo systemctl is-active --quiet shiboroom-backend; then
    echo "✅ バックエンド起動成功"
else
    echo "❌ バックエンド起動失敗"
    sudo journalctl -u shiboroom-backend -n 20 --no-pager
fi

echo ""
echo "=========================================="
echo "✅ Meilisearchキー修正完了"
echo "=========================================="
echo ""
echo "新しいマスターキー: $NEW_KEY"
echo ""
echo "動作確認:"
echo "  curl http://localhost:7700/health"
echo "  curl https://shiboroom.com/api/properties"
echo ""
