#!/usr/bin/env bash
# cronで定期スクレイピングを設定
# サーバー上で実行: bash /tmp/setup-cron-scraping.sh

set -euo pipefail

echo "==== スクレイピング定期実行の設定 ===="
echo ""

# スクリプトをサーバー上の永続的な場所にコピー
echo "[1/4] スクレイピングスクリプトを配置..."
cp /tmp/scrape-tokyo-23.sh /var/www/shiboroom/scrape-tokyo-23.sh
chmod +x /var/www/shiboroom/scrape-tokyo-23.sh

# ログディレクトリを作成
echo "[2/4] ログディレクトリを作成..."
mkdir -p /var/www/shiboroom/logs

# 5分後に実行するための一時的なcron設定
CURRENT_TIME=$(date '+%Y-%m-%d %H:%M:%S')
FUTURE_TIME=$(date -d '+5 minutes' '+%H:%M')
FUTURE_DATE=$(date -d '+5 minutes' '+%Y-%m-%d')

echo "[3/4] 5分後（${FUTURE_TIME}）に実行するcronを設定..."

# 既存のcronを保存
crontab -l > /tmp/current_cron 2>/dev/null || echo "" > /tmp/current_cron

# 5分後に1回だけ実行する設定を追加
MINUTE=$(date -d '+5 minutes' '+%M')
HOUR=$(date -d '+5 minutes' '+%H')

# 既存のscrape-tokyo-23の設定を削除
grep -v "scrape-tokyo-23.sh" /tmp/current_cron > /tmp/new_cron || echo "" > /tmp/new_cron

# 新しい設定を追加（5分後に1回実行）
echo "${MINUTE} ${HOUR} * * * cd /var/www/shiboroom && bash scrape-tokyo-23.sh >> logs/scraping.log 2>&1" >> /tmp/new_cron

# cronを更新
crontab /tmp/new_cron

echo "[4/4] 設定完了"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ スクレイピングスケジュール設定完了"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "実行予定:"
echo "  日時: ${FUTURE_TIME} (約5分後)"
echo "  スクリプト: /var/www/shiboroom/scrape-tokyo-23.sh"
echo "  ログ: /var/www/shiboroom/logs/scraping.log"
echo ""
echo "現在のcron設定:"
crontab -l | grep scrape || echo "  (なし)"
echo ""
echo "ログ確認方法:"
echo "  tail -f /var/www/shiboroom/logs/scraping.log"
echo ""
echo "cron設定確認:"
echo "  crontab -l"
echo ""
echo "定期実行を停止する場合:"
echo "  crontab -e で該当行を削除"
echo ""
