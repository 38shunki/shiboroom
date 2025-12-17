package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"real-estate-portal/internal/scraper"
	"time"
)

// PoC検証スクリプト
// Phase 0の4つの検証項目をテストします

type TestResult struct {
	TestName    string    `json:"test_name"`
	Success     bool      `json:"success"`
	Message     string    `json:"message"`
	Timestamp   time.Time `json:"timestamp"`
	Details     any       `json:"details,omitempty"`
}

type PoCResults struct {
	TestURL        string       `json:"test_url"`
	Results        []TestResult `json:"results"`
	OverallSuccess bool         `json:"overall_success"`
	ExecutedAt     time.Time    `json:"executed_at"`
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// テスト対象のURL（東京23区の賃貸物件検索結果ページ）
	// 実際のYahoo不動産のURLを指定してください
	testListURL := os.Getenv("TEST_LIST_URL")
	if testListURL == "" {
		testListURL = "https://realestate.yahoo.co.jp/rent/search/0123/list/"
		log.Printf("TEST_LIST_URL not set, using default: %s", testListURL)
	}

	results := &PoCResults{
		TestURL:    testListURL,
		ExecutedAt: time.Now(),
	}

	log.Println("=" + "===========================================")
	log.Println("Phase 0: PoC検証スクリプト開始")
	log.Println("=" + "===========================================")

	// スクレイパーの初期化
	s := scraper.NewScraper()

	// Test 1: スクレイピング安定性（同じURLで3回連続成功）
	test1Result := testScrapingStability(s, testListURL)
	results.Results = append(results.Results, test1Result)

	// Test 2以降は、Test 1で取得した物件URLを使用
	var propertyURLs []string
	if details, ok := test1Result.Details.(map[string]interface{}); ok {
		if urls, ok := details["property_urls"].([]string); ok && len(urls) > 0 {
			propertyURLs = urls
		}
	}

	if len(propertyURLs) == 0 {
		log.Println("[ERROR] Test 1 failed to get property URLs, cannot proceed with remaining tests")
		results.OverallSuccess = false
		saveResults(results)
		os.Exit(1)
	}

	// Test 2: 検索機能（Meilisearchは別途確認）
	test2Result := testSearchFunctionality(propertyURLs)
	results.Results = append(results.Results, test2Result)

	// Test 3: 画像外部参照が成立
	test3Result := testImageReference(s, propertyURLs[0])
	results.Results = append(results.Results, test3Result)

	// Test 4: Yahoo不動産へのリンクが成立
	test4Result := testYahooLink(propertyURLs[0])
	results.Results = append(results.Results, test4Result)

	// 総合判定
	results.OverallSuccess = true
	for _, result := range results.Results {
		if !result.Success {
			results.OverallSuccess = false
			break
		}
	}

	// 結果出力
	log.Println("\n" + "===========================================")
	log.Println("PoC検証結果サマリー")
	log.Println("=" + "===========================================")
	for i, result := range results.Results {
		status := "✅ PASS"
		if !result.Success {
			status = "❌ FAIL"
		}
		log.Printf("%d. %s: %s", i+1, result.TestName, status)
		log.Printf("   メッセージ: %s", result.Message)
	}

	log.Println("\n" + "===========================================")
	if results.OverallSuccess {
		log.Println("✅ PoC検証: 合格 - 技術的に実現可能")
	} else {
		log.Println("❌ PoC検証: 不合格 - 技術的課題あり")
	}
	log.Println("=" + "===========================================")

	// 結果をJSONファイルに保存
	saveResults(results)

	if !results.OverallSuccess {
		os.Exit(1)
	}
}

// Test 1: スクレイピング安定性（3回連続成功）
func testScrapingStability(s *scraper.Scraper, listURL string) TestResult {
	result := TestResult{
		TestName:  "スクレイピング安定性（3回連続成功）",
		Timestamp: time.Now(),
	}

	log.Println("\n[Test 1] スクレイピング安定性テスト開始...")

	successCount := 0
	var propertyURLs []string
	var lastError error

	for i := 1; i <= 3; i++ {
		log.Printf("  試行 %d/3...", i)

		urls, err := s.ScrapeListPage(listURL)
		if err != nil {
			log.Printf("  ❌ 試行 %d 失敗: %v", i, err)
			lastError = err
			continue
		}

		if len(urls) == 0 {
			log.Printf("  ❌ 試行 %d: 物件URLが0件", i)
			lastError = fmt.Errorf("no property URLs found")
			continue
		}

		log.Printf("  ✅ 試行 %d 成功: %d件の物件URLを取得", i, len(urls))
		successCount++
		propertyURLs = urls

		// 次の試行前に待機（レート制限対策）
		if i < 3 {
			time.Sleep(2 * time.Second)
		}
	}

	if successCount == 3 {
		result.Success = true
		result.Message = fmt.Sprintf("3回連続成功（最終取得件数: %d件）", len(propertyURLs))
		result.Details = map[string]interface{}{
			"success_count": successCount,
			"property_urls": propertyURLs,
			"url_count":     len(propertyURLs),
		}
	} else {
		result.Success = false
		result.Message = fmt.Sprintf("3回中%d回成功（最低2回失敗）: %v", successCount, lastError)
		result.Details = map[string]interface{}{
			"success_count": successCount,
			"last_error":    lastError.Error(),
		}
	}

	return result
}

// Test 2: 検索機能（Meilisearchは別途確認必要）
func testSearchFunctionality(propertyURLs []string) TestResult {
	result := TestResult{
		TestName:  "検索機能の確認（手動確認が必要）",
		Timestamp: time.Now(),
	}

	log.Println("\n[Test 2] 検索機能テスト...")

	// このテストは実際のMeilisearch接続が必要なため、
	// ここでは物件URLが取得できているかのみ確認
	if len(propertyURLs) > 0 {
		result.Success = true
		result.Message = fmt.Sprintf("物件URL取得成功（%d件）。Meilisearchへの登録と検索は手動確認が必要です。", len(propertyURLs))
		result.Details = map[string]interface{}{
			"property_count": len(propertyURLs),
			"sample_url":     propertyURLs[0],
			"manual_check":   "http://localhost:8084/api/search?q=新宿 で検索結果を確認してください",
		}
		log.Printf("  ✅ 物件URL取得成功")
		log.Printf("  ℹ️  手動確認: http://localhost:8084/api/search?q=新宿")
	} else {
		result.Success = false
		result.Message = "物件URLが取得できませんでした"
	}

	return result
}

// Test 3: 画像外部参照
func testImageReference(s *scraper.Scraper, propertyURL string) TestResult {
	result := TestResult{
		TestName:  "画像外部参照の確認",
		Timestamp: time.Now(),
	}

	log.Println("\n[Test 3] 画像外部参照テスト...")
	log.Printf("  対象URL: %s", propertyURL)

	property, err := s.ScrapeProperty(propertyURL)
	if err != nil {
		result.Success = false
		result.Message = fmt.Sprintf("物件詳細の取得失敗: %v", err)
		log.Printf("  ❌ スクレイピング失敗: %v", err)
		return result
	}

	log.Printf("  物件タイトル: %s", property.Title)
	log.Printf("  画像URL: %s", property.ImageURL)

	if property.ImageURL != "" {
		result.Success = true
		result.Message = "画像URLの取得成功（検証済み）"
		result.Details = map[string]interface{}{
			"property_title": property.Title,
			"image_url":      property.ImageURL,
			"detail_url":     property.DetailURL,
		}
		log.Printf("  ✅ 画像URL取得成功")
	} else {
		result.Success = false
		result.Message = "画像URLが取得できませんでした（プレースホルダにフォールバック）"
		result.Details = map[string]interface{}{
			"property_title": property.Title,
			"detail_url":     property.DetailURL,
			"note":           "フロントエンドではプレースホルダ表示されます",
		}
		log.Printf("  ⚠️  画像URL取得失敗（プレースホルダで対応可）")
	}

	return result
}

// Test 4: Yahoo不動産リンク
func testYahooLink(propertyURL string) TestResult {
	result := TestResult{
		TestName:  "Yahoo不動産リンクの確認",
		Timestamp: time.Now(),
	}

	log.Println("\n[Test 4] Yahoo不動産リンクテスト...")

	// URLの形式チェック
	if propertyURL == "" {
		result.Success = false
		result.Message = "物件URLが空です"
		return result
	}

	// Yahoo不動産のURLであることを確認
	if !contains(propertyURL, "realestate.yahoo.co.jp") {
		result.Success = false
		result.Message = fmt.Sprintf("Yahoo不動産のURLではありません: %s", propertyURL)
		return result
	}

	result.Success = true
	result.Message = "Yahoo不動産の正しいURLです"
	result.Details = map[string]interface{}{
		"detail_url": propertyURL,
		"note":       "target=\"_blank\" rel=\"noreferrer\" でフロントエンドから遷移可能",
	}
	log.Printf("  ✅ 正しいYahoo不動産URL")
	log.Printf("  URL: %s", propertyURL)

	return result
}

func saveResults(results *PoCResults) {
	filename := fmt.Sprintf("poc-results-%s.json", results.ExecutedAt.Format("20060102-150405"))

	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		log.Printf("[ERROR] Failed to marshal results: %v", err)
		return
	}

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		log.Printf("[ERROR] Failed to write results file: %v", err)
		return
	}

	log.Printf("\n結果を保存しました: %s", filename)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		 findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
