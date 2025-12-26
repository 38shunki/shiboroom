#!/usr/bin/env bash
# 新宿区の物件を定期的にキューへ投入（詳細スクレイピングはworkerが5件/時で処理）
# 安全設計：リスト取得のみ、詳細は増やさない、WAF対策万全

set -euo pipefail

LOCK=/tmp/enqueue_shinjuku.lock
exec 9>"$LOCK"
flock -n 9 || exit 0  # 多重起動防止

# 機械的周期回避（0〜240秒のランダムジッター）
sleep $((RANDOM % 241))

BASE="https://realestate.yahoo.co.jp/rent/search/03/13/13104/"
API="http://localhost:8085/api/scrape/list"

UA="Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"

LOG_DIR="/var/www/shiboroom/logs"
LOG_FILE="$LOG_DIR/enqueue_shinjuku.log"
mkdir -p "$LOG_DIR"

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG_FILE"
}

log "========================================"
log "新宿区キュー投入開始"
log "========================================"

# 1回で入れすぎない（各ページlimit=50、3ページまで）
TOTAL_ENQUEUED=0
TOTAL_FAILED=0

for page in 1 2 3; do
    url="${BASE}?page=${page}"
    log "ページ${page}を処理中: ${url}"

    # API呼び出し（タイムアウト60秒）
    response=$(timeout 60 curl -sS -X POST "$API" \
        -H "Content-Type: application/json" \
        -H "User-Agent: $UA" \
        -d "{\"url\":\"$url\",\"limit\":50}" 2>&1 || echo '{"error":"timeout or curl failed"}')

    # レスポンスチェック
    if echo "$response" | grep -q '"urls_found"'; then
        urls_found=$(echo "$response" | grep -o '"urls_found":[0-9]*' | grep -o '[0-9]*' || echo "0")
        new_to_queue=$(echo "$response" | grep -o '"new_to_queue":[0-9]*' | grep -o '[0-9]*' || echo "0")
        log "  ✅ ${urls_found}件の物件URLを発見（新規キュー投入: ${new_to_queue}件）"
        TOTAL_ENQUEUED=$((TOTAL_ENQUEUED + new_to_queue))
    else
        log "  ❌ エラー発生: $response"
        TOTAL_FAILED=$((TOTAL_FAILED + 1))
    fi

    # リスト側も軽く間隔（8〜15秒）
    sleep $((8 + RANDOM % 8))
done

log "========================================"
log "完了: 投入=${TOTAL_ENQUEUED}件, 失敗=${TOTAL_FAILED}ページ"
log "========================================"
log ""

exit 0
