#!/usr/bin/env bash
# 物件情報スクレイピングスクリプト
# サーバー上で実行: bash /var/www/shiboroom/scrape-properties.sh [URL]

set -euo pipefail

API_URL="https://shiboroom.com"

# 使い方表示
show_usage() {
    echo "使い方:"
    echo "  bash $0 [URL]"
    echo ""
    echo "例:"
    echo "  bash $0 https://realestate.yahoo.co.jp/rent/search/..."
    echo ""
    echo "URLを指定しない場合は、対話式で入力できます。"
    exit 1
}

echo "==== Shiboroom 物件スクレイピング ===="
echo ""

# URLを引数から取得、なければ対話式
if [ $# -eq 0 ]; then
    echo "スクレイピングしたい物件のURLを入力してください:"
    read -r SCRAPE_URL
else
    SCRAPE_URL="$1"
fi

# URLが空でないか確認
if [ -z "$SCRAPE_URL" ]; then
    echo "❌ エラー: URLが指定されていません"
    show_usage
fi

echo "対象URL: $SCRAPE_URL"
echo ""

# スクレイピング実行
echo "[1/3] スクレイピングを実行中..."
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/api/scrape" \
    -H "Content-Type: application/json" \
    -d "{\"url\":\"$SCRAPE_URL\"}")

# HTTPステータスコードとレスポンスボディを分離
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')

echo ""
if [ "$HTTP_CODE" = "200" ]; then
    echo "✅ スクレイピング成功！"
    echo ""
    echo "レスポンス:"
    echo "$BODY" | python3 -m json.tool 2>/dev/null || echo "$BODY"

    # データベース確認
    echo ""
    echo "[2/3] データベースの物件数を確認..."
    PROPERTIES_RESPONSE=$(curl -s "$API_URL/api/properties")
    PROPERTY_COUNT=$(echo "$PROPERTIES_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); print(len(data))" 2>/dev/null || echo "不明")
    echo "現在の物件数: $PROPERTY_COUNT 件"

    # Meilisearch確認
    echo ""
    echo "[3/3] 検索インデックスを確認..."
    SEARCH_RESPONSE=$(curl -s -X POST "$API_URL/api/search/advanced" \
        -H "Content-Type: application/json" \
        -d '{"query":"","limit":100}')
    SEARCH_COUNT=$(echo "$SEARCH_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('total_hits', 0))" 2>/dev/null || echo "不明")
    echo "検索インデックスの物件数: $SEARCH_COUNT 件"

    echo ""
    echo "=========================================="
    echo "✅ スクレイピング完了"
    echo "=========================================="
    echo ""
    echo "🌐 ブラウザで確認:"
    echo "  $API_URL"
    echo ""

else
    echo "❌ スクレイピング失敗 (HTTP $HTTP_CODE)"
    echo ""
    echo "エラー内容:"
    echo "$BODY"
    echo ""
    echo "トラブルシューティング:"
    echo "  1. バックエンドログ確認: sudo journalctl -u shiboroom-backend -n 50"
    echo "  2. バックエンドステータス: sudo systemctl status shiboroom-backend"
    echo "  3. URLが正しいか確認してください"
    exit 1
fi
