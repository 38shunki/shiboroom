#!/usr/bin/env bash
# テスト用スクレイピング（千代田区1ページのみ）
set -euo pipefail

API_URL="http://localhost:8085"

echo "==== テストスクレイピング開始 ===="
echo "対象: 千代田区 1ページのみ"
echo ""

LIST_URL="https://realestate.yahoo.co.jp/rent/search/03/13/13101/?page=1"

echo "📍 千代田区"
echo -n "  [p1] "

# API呼び出し (concurrency=3で並列処理)
RESPONSE=$(curl -s -X POST "${API_URL}/api/scrape/list" \
    -H "Content-Type: application/json" \
    -d "{\"url\":\"${LIST_URL}\",\"concurrency\":3}" 2>&1)

# レスポンス確認
if echo "$RESPONSE" | grep -q "scraped\|found"; then
    SCRAPED=$(echo "$RESPONSE" | grep -o '"scraped":[0-9]*' | grep -o '[0-9]*' || echo "0")
    FAILED=$(echo "$RESPONSE" | grep -o '"failed":[0-9]*' | grep -o '[0-9]*' || echo "0")

    echo "✅ +${SCRAPED} / 失敗:${FAILED}"
    echo ""
    echo "詳細:"
    echo "$RESPONSE" | python3 -m json.tool 2>/dev/null | head -30
else
    echo "❌ エラー"
    echo "$RESPONSE"
fi

echo ""
echo "==== テスト完了 ===="
