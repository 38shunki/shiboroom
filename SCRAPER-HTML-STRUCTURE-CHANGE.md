# Yahoo不動産 HTMLパーサー修正ガイド

## 問題：スクレイパーが0件を返す

### 発生日時
2025-12-18

### 症状
- `/api/scrape/list` APIが0件の物件を返す
- ログに `[ScrapeListPage] Found 0 unique property URLs` と表示される

### 原因
Yahoo不動産がHTML構造を変更し、物件IDの格納場所が変わった：

**変更前**：
- `<a href="/rent/detail/[48文字のhex ID]">` のような直接リンクがHTMLに存在

**変更後**：
- 直接リンクは存在しない（JavaScriptで動的生成）
- 物件IDは `<input class="_propertyCheckbox" value="[48文字のhex ID]">` のcheckbox要素のvalue属性に格納

### HTML構造の調査方法

1. **curlでHTMLを取得**
   ```bash
   curl -sSL \
     -A "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36" \
     -H "Accept-Language: ja,en-US;q=0.9,en;q=0.8" \
     "https://realestate.yahoo.co.jp/rent/search/03/13/13101/?page=1" \
     -o /tmp/yahoo_list.html
   ```

2. **物件IDの存在確認**
   ```bash
   # 旧方式（a要素内のリンク）を確認
   grep -oE "/rent/detail/[0-9a-f]{48}" /tmp/yahoo_list.html

   # 新方式（checkbox value）を確認
   grep -A 2 "_propertyCheckbox\"" /tmp/yahoo_list.html | grep "value="
   ```

3. **HTMLサイズ確認**
   ```bash
   wc -c /tmp/yahoo_list.html  # 1.7MB程度が正常
   ```

### 修正内容

**ファイル**: `backend/internal/scraper/scraper.go`

**変更箇所**: `ScrapeListPage` 関数 (288-312行目)

**修正前のコード**:
```go
// Find all links that point to property detail pages
// Yahoo Real Estate detail URLs follow the pattern: /rent/detail/
doc.Find("a").Each(func(i int, s *goquery.Selection) {
    if href, exists := s.Attr("href"); exists {
        // Check if it's a property detail URL
        if !strings.Contains(href, "/rent/detail/") {
            return
        }

        // Convert relative URL to absolute
        propertyURL := href
        if strings.HasPrefix(href, "/") {
            propertyURL = "https://realestate.yahoo.co.jp" + href
        } else if !strings.HasPrefix(href, "http") {
            // Skip invalid URLs
            return
        }

        // Normalize URL to avoid duplicates
        normalizedURL := normalizeURL(propertyURL)

        // Add only unique URLs
        if !seenURLs[normalizedURL] {
            seenURLs[normalizedURL] = true
            propertyURLs = append(propertyURLs, normalizedURL)
        }
    }
})
```

**修正後のコード**:
```go
// Find all property checkboxes (Yahoo changed HTML structure - property IDs are now in checkbox values)
// Property IDs are 48-character hex strings in input._propertyCheckbox value attributes
doc.Find("input._propertyCheckbox").Each(func(i int, s *goquery.Selection) {
    if value, exists := s.Attr("value"); exists && len(value) >= 48 {
        // Remove leading underscore if present
        propertyID := strings.TrimPrefix(value, "_")

        // Extract first 48 characters (hex ID)
        if len(propertyID) >= 48 {
            propertyID = propertyID[:48]
        }

        // Build detail URL
        propertyURL := "https://realestate.yahoo.co.jp/rent/detail/" + propertyID

        // Normalize URL to avoid duplicates
        normalizedURL := normalizeURL(propertyURL)

        // Add only unique URLs
        if !seenURLs[normalizedURL] {
            seenURLs[normalizedURL] = true
            propertyURLs = append(propertyURLs, normalizedURL)
        }
    }
})
```

### 主な変更点

1. **セレクタ変更**: `doc.Find("a")` → `doc.Find("input._propertyCheckbox")`
2. **ID抽出**: href属性から → value属性から
3. **先頭アンダースコア対応**: 一部のIDは `_000006835304...` のように先頭に `_` が付いているため、`strings.TrimPrefix` で削除
4. **48文字抽出**: value属性の最初の48文字をIDとして抽出
5. **URL構築**: 抽出したIDから詳細URLを構築

### デプロイ手順

1. **ローカルでビルド**
   ```bash
   cd /Users/shu/Documents/dev/real-estate-portal/backend
   docker run --rm --platform linux/amd64 -v "$(pwd)":/app -w /app golang:1.23-alpine sh -c \
     "apk add --no-cache gcc musl-dev >/dev/null 2>&1 && \
      CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
      go build -ldflags '-extldflags \"-static\"' -o shiboroom-api ./cmd/api"
   ```

2. **サーバーにアップロード**
   ```bash
   scp shiboroom-api grik@162.43.74.38:/tmp/shiboroom-api-new
   ```

3. **サーバーで配置**
   ```bash
   ssh grik@162.43.74.38
   sudo mv /tmp/shiboroom-api-new /var/www/shiboroom/backend/shiboroom-api
   sudo chown grik:grik /var/www/shiboroom/backend/shiboroom-api
   sudo chmod +x /var/www/shiboroom/backend/shiboroom-api
   sudo systemctl restart shiboroom-backend
   systemctl status shiboroom-backend
   ```

### テスト方法

1. **一覧ページスクレイプテスト**
   ```bash
   curl -X POST http://localhost:8085/api/scrape/list \
     -H "Content-Type: application/json" \
     -d '{"url":"https://realestate.yahoo.co.jp/rent/search/03/13/13101/?page=1","concurrency":1,"limit":2}'
   ```

   期待される結果：
   ```json
   {
     "found": 2,
     "scraped": 2,
     "existing": 0,
     "new": 2,
     "failed": 0
   }
   ```

2. **ログ確認**
   ```bash
   sudo journalctl -u shiboroom-backend -n 50 --no-pager | grep "ScrapeListPage"
   ```

   期待されるログ：
   ```
   [ScrapeListPage] Found 2 unique property URLs from https://...
   ```

### 予防策

将来的にHTML構造が再度変更される可能性を考慮し、以下の対策を推奨：

1. **正規表現フォールバック**
   HTML全体から `/rent/detail/[48-hex]` パターンを正規表現で抽出する方法を追加：
   ```go
   var reDetail = regexp.MustCompile(`/rent/detail/([0-9a-f]{48})`)

   func extractDetailURLsByRegex(html string) []string {
       m := reDetail.FindAllStringSubmatch(html, -1)
       seen := map[string]struct{}{}
       out := make([]string, 0, len(m))
       for _, mm := range m {
           id := mm[1]
           u := "https://realestate.yahoo.co.jp/rent/detail/" + id
           if _, ok := seen[u]; ok { continue }
           seen[u] = struct{}{}
           out = append(out, u)
       }
       return out
   }
   ```

2. **モニタリング**
   - 定期的に `[ScrapeListPage] Found 0 unique property URLs` ログをチェック
   - 0件が続く場合は HTML構造変更の可能性を疑う

3. **アラート**
   - スクレイプ成功率が閾値を下回った場合に通知

### 関連ドキュメント
- `DEPLOYMENT-TROUBLESHOOTING.md`: デプロイ時の問題解決ガイド
- `backend/internal/scraper/scraper.go`: スクレイパー本体

### 更新履歴
- **2025-12-18**: 初版作成 - checkbox value属性からのID抽出に対応
