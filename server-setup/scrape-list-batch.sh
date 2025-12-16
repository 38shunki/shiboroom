#!/usr/bin/env bash
# 一覧ページから物件を一括スクレイピング
# サーバー上で実行: bash /var/www/shiboroom/scrape-list-batch.sh [LIST_URL] [MAX_PAGES]

set -euo pipefail

API_URL="https://shiboroom.com"

# 使い方表示
show_usage() {
    echo "使い方:"
    echo "  bash $0 [LIST_URL] [MAX_PAGES]"
    echo ""
    echo "例:"
    echo "  bash $0 'https://realestate.yahoo.co.jp/rent/search/...' 10"
    echo ""
    echo "パラメータ:"
    echo "  LIST_URL  : 検索結果一覧ページのURL"
    echo "  MAX_PAGES : スクレイピングする最大ページ数（省略時は1ページ）"
    echo ""
    exit 1
}

echo "==== Shiboroom 一括物件スクレイピング ===="
echo ""

# URLを引数から取得
if [ $# -eq 0 ]; then
    echo "検索結果一覧ページのURLを入力してください:"
    read -r LIST_URL
    echo "スクレイピングする最大ページ数を入力してください（Enter=1ページ）:"
    read -r MAX_PAGES
    MAX_PAGES=${MAX_PAGES:-1}
else
    LIST_URL="$1"
    MAX_PAGES="${2:-1}"
fi

# URLが空でないか確認
if [ -z "$LIST_URL" ]; then
    echo "❌ エラー: URLが指定されていません"
    show_usage
fi

echo "一覧ページURL: $LIST_URL"
echo "最大ページ数: $MAX_PAGES"
echo ""

TOTAL_SCRAPED=0
TOTAL_FAILED=0

# ページごとにスクレイピング
for ((page=1; page<=MAX_PAGES; page++)); do
    echo "=========================================="
    echo "📄 ページ $page / $MAX_PAGES をスクレイピング中..."
    echo "=========================================="

    # ページ番号をURLに追加
    # Yahoo不動産の場合: ?page=N というクエリパラメータ
    if [[ "$LIST_URL" == *"?"* ]]; then
        # 既にクエリパラメータがある場合
        CURRENT_URL="${LIST_URL}&page=${page}"
    else
        # クエリパラメータがない場合
        CURRENT_URL="${LIST_URL}?page=${page}"
    fi

    echo "URL: $CURRENT_URL"
    echo ""

    # 一覧ページから物件URLを取得してスクレイピング
    echo "[1/2] 一覧ページから物件URLを抽出..."
    RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/api/scrape/list" \
        -H "Content-Type: application/json" \
        -d "{\"url\":\"$CURRENT_URL\"}")

    HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
    BODY=$(echo "$RESPONSE" | sed '$d')

    if [ "$HTTP_CODE" != "200" ]; then
        echo "❌ 一覧ページの取得失敗 (HTTP $HTTP_CODE)"
        echo "エラー: $BODY"
        TOTAL_FAILED=$((TOTAL_FAILED + 1))
        continue
    fi

    # 取得した物件URL数を表示
    PROPERTY_COUNT=$(echo "$BODY" | python3 -c "import sys, json; data=json.load(sys.stdin); print(len(data.get('property_urls', [])))" 2>/dev/null || echo "0")
    echo "✅ $PROPERTY_COUNT 件の物件URLを取得"

    if [ "$PROPERTY_COUNT" = "0" ]; then
        echo "⚠️  物件が見つかりませんでした。最終ページに到達した可能性があります。"
        break
    fi

    # 各物件をスクレイピング
    echo "[2/2] 各物件の詳細をスクレイピング..."
    PROPERTY_URLS=$(echo "$BODY" | python3 -c "import sys, json; data=json.load(sys.stdin); print('\n'.join(data.get('property_urls', [])))" 2>/dev/null)

    PAGE_SCRAPED=0
    PAGE_FAILED=0

    while IFS= read -r property_url; do
        if [ -z "$property_url" ]; then
            continue
        fi

        echo "  スクレイピング中: $property_url"
        PROP_RESPONSE=$(curl -s -w "%{http_code}" -X POST "$API_URL/api/scrape" \
            -H "Content-Type: application/json" \
            -d "{\"url\":\"$property_url\"}" \
            -o /dev/null)

        if [ "$PROP_RESPONSE" = "200" ]; then
            PAGE_SCRAPED=$((PAGE_SCRAPED + 1))
            echo "  ✅ 成功"
        else
            PAGE_FAILED=$((PAGE_FAILED + 1))
            echo "  ❌ 失敗 (HTTP $PROP_RESPONSE)"
        fi

        # レート制限のため少し待機
        sleep 2
    done <<< "$PROPERTY_URLS"

    TOTAL_SCRAPED=$((TOTAL_SCRAPED + PAGE_SCRAPED))
    TOTAL_FAILED=$((TOTAL_FAILED + PAGE_FAILED))

    echo ""
    echo "ページ $page 完了: 成功 $PAGE_SCRAPED 件, 失敗 $PAGE_FAILED 件"
    echo ""

    # 次のページへ進む前に少し待機
    if [ $page -lt $MAX_PAGES ]; then
        echo "次のページへ進みます（3秒待機）..."
        sleep 3
    fi
done

# 最終結果
echo "=========================================="
echo "✅ 一括スクレイピング完了"
echo "=========================================="
echo ""
echo "📊 結果サマリー:"
echo "  処理ページ数: $page ページ"
echo "  成功: $TOTAL_SCRAPED 件"
echo "  失敗: $TOTAL_FAILED 件"
echo ""

# データベース確認
echo "[確認] データベースの物件数..."
PROPERTIES_RESPONSE=$(curl -s "$API_URL/api/properties")
PROPERTY_COUNT=$(echo "$PROPERTIES_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); print(len(data))" 2>/dev/null || echo "不明")
echo "現在の総物件数: $PROPERTY_COUNT 件"

# Meilisearch確認
echo ""
echo "[確認] 検索インデックス..."
SEARCH_RESPONSE=$(curl -s -X POST "$API_URL/api/search/advanced" \
    -H "Content-Type: application/json" \
    -d '{"query":"","limit":1}')
SEARCH_COUNT=$(echo "$SEARCH_RESPONSE" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('total_hits', 0))" 2>/dev/null || echo "不明")
echo "検索インデックスの物件数: $SEARCH_COUNT 件"

echo ""
echo "🌐 ブラウザで確認:"
echo "  $API_URL"
echo ""
