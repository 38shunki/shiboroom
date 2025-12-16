#!/usr/bin/env bash
# Meilisearchのインストールとセットアップ
# サーバー上で実行: bash /tmp/setup-meilisearch.sh

set -euo pipefail

echo "==== Meilisearch セットアップ ===="
echo ""

# Meilisearchが既にインストールされているか確認
if command -v meilisearch &> /dev/null; then
    echo "✅ Meilisearchは既にインストール済み"
    MEILI_VERSION=$(meilisearch --version || echo "unknown")
    echo "バージョン: $MEILI_VERSION"
else
    echo "[1/4] Meilisearchをインストール..."

    # 最新版をダウンロード
    curl -L https://install.meilisearch.com | sh

    # バイナリを /usr/local/bin に移動
    sudo mv ./meilisearch /usr/local/bin/
    sudo chmod +x /usr/local/bin/meilisearch

    echo "✅ Meilisearchインストール完了"
fi

# systemdサービスファイルを作成
echo "[2/4] Meilisearch systemdサービスを作成..."
sudo tee /etc/systemd/system/meilisearch.service > /dev/null <<'EOF'
[Unit]
Description=Meilisearch
After=network.target

[Service]
Type=simple
User=grik
Group=grik
WorkingDirectory=/var/www/shiboroom
ExecStart=/usr/local/bin/meilisearch --http-addr 127.0.0.1:7700 --master-key masterKey123 --db-path /var/www/shiboroom/meilisearch-data

# 環境変数
Environment="MEILI_ENV=production"
Environment="MEILI_HTTP_ADDR=127.0.0.1:7700"
Environment="MEILI_MASTER_KEY=masterKey123"

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

# データディレクトリ作成
echo "[3/4] データディレクトリを作成..."
mkdir -p /var/www/shiboroom/meilisearch-data
chown -R grik:grik /var/www/shiboroom/meilisearch-data

# サービスを有効化して起動
echo "[4/4] Meilisearchサービスを起動..."
sudo systemctl daemon-reload
sudo systemctl enable meilisearch
sudo systemctl restart meilisearch

# 起動確認
sleep 3
sudo systemctl status meilisearch --no-pager || true

echo ""
echo "✅ Meilisearchセットアップ完了"
echo ""
echo "動作確認:"
echo "  curl http://localhost:7700/health"
echo ""
echo "ステータス確認:"
echo "  sudo systemctl status meilisearch"
echo ""
echo "ログ確認:"
echo "  sudo journalctl -u meilisearch -f"
echo ""
