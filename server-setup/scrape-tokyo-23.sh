#!/usr/bin/env bash
# 東京23区の物件を一括スクレイピング
# サーバー上で実行: bash /tmp/scrape-tokyo-23.sh

set -euo pipefail

API_URL="http://localhost:8085"

echo "==== 東京23区 物件スクレイピング ===="
echo "目標: 1,000件程度"
echo ""

# 東京23区の検索URL（区コード）
# https://realestate.yahoo.co.jp/rent/search/03/13/区コード/

TOKYO_23_WARDS=(
    "13101:千代田区"
    "13102:中央区"
    "13103:港区"
    "13104:新宿区"
    "13105:文京区"
    "13106:台東区"
    "13107:墨田区"
    "13108:江東区"
    "13109:品川区"
    "13110:目黒区"
    "13111:大田区"
    "13112:世田谷区"
    "13113:渋谷区"
    "13114:中野区"
    "13115:杉並区"
    "13116:豊島区"
    "13117:北区"
    "13118:荒川区"
    "13119:板橋区"
    "13120:練馬区"
    "13121:足立区"
    "13122:葛飾区"
    "13123:江戸川区"
)

TOTAL_SCRAPED=0
TOTAL_FAILED=0
MAX_PROPERTIES=1000
PAGES_PER_WARD=2  # 各区から2ページ（約40-60件）

echo "各区から${PAGES_PER_WARD}ページずつスクレイピングします"
echo "推定: 23区 × ${PAGES_PER_WARD}ページ × 約25件/ページ = 約1,150件"
echo ""

for ward_info in "${TOKYO_23_WARDS[@]}"; do
    ward_code="${ward_info%%:*}"
    ward_name="${ward_info##*:}"

    echo ""
    echo "📍 ${ward_name}"

    # 各区の検索URLを順番に処理
    for page in $(seq 1 $PAGES_PER_WARD); do
        # 目標件数に達したら終了
        if [ $TOTAL_SCRAPED -ge $MAX_PROPERTIES ]; then
            echo ""
            echo "✅ 目標件数 ${MAX_PROPERTIES}件 に達しました"
            break 2
        fi

        LIST_URL="https://realestate.yahoo.co.jp/rent/search/03/13/${ward_code}/?page=${page}"

        echo -n "  [p${page}] "

        # API呼び出し (concurrency=3で並列処理 - Yahoo側の制限対策)
        RESPONSE=$(curl -s -X POST "${API_URL}/api/scrape/list" \
            -H "Content-Type: application/json" \
            -d "{\"url\":\"${LIST_URL}\",\"concurrency\":3}" 2>&1)

        # レスポンス確認
        if echo "$RESPONSE" | grep -q "scraped\|found"; then
            SCRAPED=$(echo "$RESPONSE" | grep -o '"scraped":[0-9]*' | grep -o '[0-9]*' || echo "0")
            FAILED=$(echo "$RESPONSE" | grep -o '"failed":[0-9]*' | grep -o '[0-9]*' || echo "0")

            TOTAL_SCRAPED=$((TOTAL_SCRAPED + SCRAPED))
            TOTAL_FAILED=$((TOTAL_FAILED + FAILED))

            echo "✅ +${SCRAPED} (計:${TOTAL_SCRAPED})"
        else
            # エラーメッセージを簡潔に表示（JSON詳細は非表示）
            echo "❌ エラー"
            TOTAL_FAILED=$((TOTAL_FAILED + 1))
        fi

        # 並列処理なので待機時間を短縮
        sleep 1
    done

    # 目標達成チェック
    if [ $TOTAL_SCRAPED -ge $MAX_PROPERTIES ]; then
        break
    fi

    # 各区の間の待機を短縮
    sleep 2
done

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📊 スクレイピング完了"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "成功: ${TOTAL_SCRAPED}件"
echo "失敗: ${TOTAL_FAILED}件"
echo ""

# 最終確認
echo "データベース内の物件数:"
mysql -u shiboroom_user -p'Kihara0725$' shiboroom -e 'SELECT COUNT(*) as total FROM properties;' 2>/dev/null || echo "※MySQL接続エラー"

echo ""
echo "✅ 完了"
echo ""
