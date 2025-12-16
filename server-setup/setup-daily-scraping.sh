#!/usr/bin/env bash
# cronで毎日定期スクレイピングを設定
# サーバー上で実行: bash /tmp/setup-daily-scraping.sh

set -euo pipefail

echo "==== 毎日定期スクレイピングの設定 ===="
echo ""

# デフォルトは毎日午前2時
DEFAULT_HOUR="2"
DEFAULT_MINUTE="0"

echo "毎日何時に実行しますか？（デフォルト: 02:00）"
read -p "時 [0-23] (Enter=2): " HOUR
read -p "分 [0-59] (Enter=0): " MINUTE

HOUR=${HOUR:-$DEFAULT_HOUR}
MINUTE=${MINUTE:-$DEFAULT_MINUTE}

echo ""
echo "[1/3] スクレイピングスクリプトを配置..."
cp /tmp/scrape-tokyo-23.sh /var/www/shiboroom/scrape-tokyo-23.sh
chmod +x /var/www/shiboroom/scrape-tokyo-23.sh

echo "[2/3] ログディレクトリを作成..."
mkdir -p /var/www/shiboroom/logs

echo "[3/3] 毎日 ${HOUR}:${MINUTE} に実行するcronを設定..."

# 既存のcronを保存
crontab -l > /tmp/current_cron 2>/dev/null || echo "" > /tmp/current_cron

# 既存のscrape-tokyo-23の設定を削除
grep -v "scrape-tokyo-23.sh" /tmp/current_cron > /tmp/new_cron || echo "" > /tmp/new_cron

# 新しい設定を追加（毎日実行）
echo "${MINUTE} ${HOUR} * * * cd /var/www/shiboroom && bash scrape-tokyo-23.sh >> logs/scraping.log 2>&1" >> /tmp/new_cron

# cronを更新
crontab /tmp/new_cron

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ 毎日定期スクレイピング設定完了"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "実行スケジュール:"
echo "  毎日 ${HOUR}:${MINUTE} (JST)"
echo "  スクリプト: /var/www/shiboroom/scrape-tokyo-23.sh"
echo "  ログ: /var/www/shiboroom/logs/scraping.log"
echo ""
echo "現在のcron設定:"
crontab -l
echo ""
echo "ログ確認方法:"
echo "  tail -f /var/www/shiboroom/logs/scraping.log"
echo ""
echo "定期実行を停止する場合:"
echo "  crontab -e で該当行を削除"
echo ""
