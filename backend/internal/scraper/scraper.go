package scraper

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/url"
	"real-estate-portal/internal/models"
	"real-estate-portal/internal/ratelimit"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var (
	// Global rate limiter for Yahoo Real Estate
	// Burst control strategy: reduce concurrent requests and increase delay
	yahooLimiter = ratelimit.NewYahooLimiter(
		1,                     // maxInFlight: 1 concurrent request (avoid burst)
		2500*time.Millisecond, // baseDelay: 2.5s base
		1500*time.Millisecond, // jitter: 0-1.5s (total: 2.5-4.0s)
	)

	// Global circuit breaker to detect WAF blocks
	// Stricter early detection to avoid prolonged blocks
	circuitBreaker = NewCircuitBreaker(
		8,              // failureThreshold: 8 failures out of 20 requests (stricter)
		1*time.Hour,    // resetTimeout: wait 1 hour before retry
	)
)

type Scraper struct {
	client           *http.Client
	maxRetries       int
	retryDelay       time.Duration
	requestDelay     time.Duration
	lastRequestTime  time.Time
}

type ScraperConfig struct {
	Timeout      time.Duration
	MaxRetries   int
	RetryDelay   time.Duration
	RequestDelay time.Duration
}

func NewScraper() *Scraper {
	return NewScraperWithConfig(ScraperConfig{
		Timeout:      30 * time.Second,  // 30s for normal page fetches
		MaxRetries:   3,                  // Retry up to 3 times
		RetryDelay:   2 * time.Second,   // Base delay for exponential backoff
		RequestDelay: 2 * time.Second,   // Minimum 2s between requests (rate limiting)
	})
}

func NewScraperWithConfig(config ScraperConfig) *Scraper {
	return &Scraper{
		client: &http.Client{
			Timeout: config.Timeout,
		},
		maxRetries:   config.MaxRetries,
		retryDelay:   config.RetryDelay,
		requestDelay: config.RequestDelay,
	}
}

// rateLimit enforces minimum delay between requests
func (s *Scraper) rateLimit() {
	if s.requestDelay == 0 {
		return
	}

	elapsed := time.Since(s.lastRequestTime)
	if elapsed < s.requestDelay {
		time.Sleep(s.requestDelay - elapsed)
	}
	s.lastRequestTime = time.Now()
}

// doRequestWithRetry performs HTTP request with exponential backoff retry
func (s *Scraper) doRequestWithRetry(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	// Check circuit breaker before proceeding
	if !circuitBreaker.CanProceed() {
		isOpen, failures, total := circuitBreaker.GetStatus()
		return nil, fmt.Errorf("circuit breaker open: suspected WAF block (%d/%d failures, open=%v)", failures, total, isOpen)
	}

	// Acquire global rate limiter before starting
	yahooLimiter.Acquire()
	defer yahooLimiter.Release()

	for attempt := 0; attempt <= s.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: delay * 2^(attempt-1), max 60s
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * s.retryDelay
			if backoff > 60*time.Second {
				backoff = 60 * time.Second
			}
			log.Printf("Retry attempt %d/%d after %v (inFlight: %d)", attempt, s.maxRetries, backoff, yahooLimiter.GetInFlight())
			time.Sleep(backoff)
		}

		resp, err = s.client.Do(req)

		if err == nil && resp.StatusCode == 200 {
			circuitBreaker.RecordSuccess()
			return resp, nil
		}

		// Log error with status code breakdown
		if err != nil {
			log.Printf("Request failed (attempt %d): %v", attempt+1, err)
			circuitBreaker.RecordFailure(0)
		} else {
			log.Printf("Request failed (attempt %d): status %d (inFlight: %d)", attempt+1, resp.StatusCode, yahooLimiter.GetInFlight())

			// Record failure for circuit breaker
			if resp.StatusCode >= 500 || resp.StatusCode == 429 || resp.StatusCode == 403 {
				circuitBreaker.RecordFailure(resp.StatusCode)
			}

			if resp.Body != nil {
				resp.Body.Close()
			}

			// Longer backoff for server errors (500/503)
			if resp.StatusCode >= 500 && attempt < s.maxRetries {
				serverBackoff := time.Duration(math.Pow(2, float64(attempt+2))) * s.retryDelay
				if serverBackoff > 60*time.Second {
					serverBackoff = 60 * time.Second
				}
				log.Printf("Server error %d, backing off for %v", resp.StatusCode, serverBackoff)
				time.Sleep(serverBackoff)
			}
		}

		// Don't retry on client errors (4xx except 429)
		if resp != nil && resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != 429 {
			break
		}
	}

	if err != nil {
		return nil, fmt.Errorf("request failed after %d retries: %w", s.maxRetries, err)
	}
	return nil, fmt.Errorf("request failed after %d retries: status code %d", s.maxRetries, resp.StatusCode)
}

// ScrapeListPage scrapes a list page and returns property URLs
func (s *Scraper) ScrapeListPage(listURL string) ([]string, error) {
	log.Printf("[ScrapeListPage] Starting scrape of list page: %s", listURL)

	req, err := http.NewRequest("GET", listURL, nil)
	if err != nil {
		log.Printf("[ScrapeListPage] Error creating request for %s: %v", listURL, err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

	resp, err := s.doRequestWithRetry(req)
	if err != nil {
		log.Printf("[ScrapeListPage] Error fetching list page %s: %v", listURL, err)
		return nil, fmt.Errorf("failed to fetch list page: %w", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Printf("[ScrapeListPage] Error parsing HTML from %s: %v", listURL, err)
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var propertyURLs []string
	seenURLs := make(map[string]bool)

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

	log.Printf("[ScrapeListPage] Found %d unique property URLs from %s", len(propertyURLs), listURL)
	return propertyURLs, nil
}

// ScrapeProperty scrapes a property detail page
func (s *Scraper) ScrapeProperty(inputURL string) (*models.Property, error) {
	// Normalize URL (remove query strings, trailing slash)
	normalizedURL := normalizeURL(inputURL)
	log.Printf("[ScrapeProperty] Starting scrape of property: %s (normalized: %s)", inputURL, normalizedURL)

	// Fetch the page
	req, err := http.NewRequest("GET", normalizedURL, nil)
	if err != nil {
		log.Printf("[ScrapeProperty] Error creating request for %s: %v", normalizedURL, err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set User-Agent to mimic a browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

	resp, err := s.doRequestWithRetry(req)
	if err != nil {
		log.Printf("[ScrapeProperty] Error fetching URL %s: %v", normalizedURL, err)
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	// Parse HTML
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Printf("[ScrapeProperty] Error parsing HTML from %s: %v", normalizedURL, err)
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Check for canonical URL
	canonicalURL := extractCanonicalURL(doc)
	if canonicalURL != "" {
		normalizedURL = normalizeURL(canonicalURL)
	}

	// Extract Yahoo property ID from URL
	yahooPropertyID, err := extractYahooPropertyID(normalizedURL)
	if err != nil {
		log.Printf("[ScrapeProperty] Warning: Could not extract Yahoo property ID from %s: %v", normalizedURL, err)
		// Fallback to URL hash for non-standard URLs
		hash := md5.Sum([]byte(normalizedURL))
		yahooPropertyID = hex.EncodeToString(hash[:])
	}

	// Extract metadata
	property := &models.Property{
		Source:           "yahoo",
		SourcePropertyID: yahooPropertyID,
		DetailURL:        normalizedURL,
		FetchedAt:        time.Now(),
	}

	// Try to get og:title
	if title, exists := doc.Find("meta[property='og:title']").Attr("content"); exists {
		property.Title = strings.TrimSpace(title)
	}

	// Fallback to page title if og:title not found
	if property.Title == "" {
		property.Title = strings.TrimSpace(doc.Find("title").Text())
	}

	// Try to extract ExternalImageUrl from JSON data embedded in the page
	// Yahoo Real Estate embeds property data in window.__SERVER_SIDE_CONTEXT__
	pageHTML, _ := doc.Html()
	externalImageURL := extractExternalImageFromJSON(pageHTML)

	if externalImageURL != "" {
		// Verify external image URL is accessible
		if s.verifyImageURL(externalImageURL) {
			property.ImageURL = externalImageURL
		}
	} else {
		// Fallback to og:image if ExternalImageUrl not found
		if imageURL, exists := doc.Find("meta[property='og:image']").Attr("content"); exists {
			imageURL = strings.TrimSpace(imageURL)
			// Verify image URL is accessible
			if s.verifyImageURL(imageURL) {
				property.ImageURL = imageURL
			}
		}
	}

	// Extract additional details from the page
	s.extractDetailFields(doc, property)

	// Generate internal ID from source + source_property_id
	// This ensures consistent ID generation across the application
	idSource := property.Source + ":" + property.SourcePropertyID
	hash := md5.Sum([]byte(idSource))
	property.ID = hex.EncodeToString(hash[:])

	// Validate required fields
	if property.Title == "" {
		property.Title = "No Title"
		log.Printf("[ScrapeProperty] Warning: No title found for %s", normalizedURL)
	}

	log.Printf("[ScrapeProperty] Successfully scraped property %s (ID: %s, Title: %s)", normalizedURL, property.ID, property.Title)
	return property, nil
}

// extractDetailFields extracts detailed property information from the DOM
func (s *Scraper) extractDetailFields(doc *goquery.Document, property *models.Property) {
	// Extract from the page text (best effort)
	pageText := doc.Text()

	// Extract rent (賃料)
	if rent := extractRent(pageText); rent > 0 {
		property.Rent = &rent
	}

	// Extract floor plan (間取り)
	property.FloorPlan = extractFloorPlan(pageText)

	// Extract area (面積)
	if area := extractArea(pageText); area > 0 {
		property.Area = &area
	}

	// Extract walk time (徒歩)
	if walkTime := extractWalkTime(pageText); walkTime > 0 {
		property.WalkTime = &walkTime
	}

	// Extract station (駅名)
	property.Station = extractStation(pageText)

	// Extract address (住所)
	property.Address = extractAddress(doc)

	// Extract building age (築年数)
	if age := extractBuildingAge(pageText); age > 0 {
		property.BuildingAge = &age
	}

	// Extract floor (階数)
	if floor := extractFloor(pageText); floor != 0 {
		property.Floor = &floor
	}
}

// extractRent extracts rent amount from text
func extractRent(text string) int {
	// Pattern: "8.5万円" or "85000円"
	re := regexp.MustCompile(`([0-9]+\.?[0-9]*)万円`)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
			rent := int(val * 10000)
			// Validate: rent should be reasonable (10,000 - 10,000,000 yen)
			if rent >= 10000 && rent <= 10000000 {
				return rent
			}
		}
	}

	// Pattern: direct yen amount
	re = regexp.MustCompile(`賃料[：:]\s*([0-9,]+)円`)
	matches = re.FindStringSubmatch(text)
	if len(matches) > 1 {
		cleaned := strings.ReplaceAll(matches[1], ",", "")
		if val, err := strconv.Atoi(cleaned); err == nil {
			// Validate: rent should be reasonable (10,000 - 10,000,000 yen)
			if val >= 10000 && val <= 10000000 {
				return val
			}
		}
	}

	return 0
}

// extractFloorPlan extracts floor plan (1K, 1DK, etc.)
func extractFloorPlan(text string) string {
	// Pattern: "1K", "1DK", "1LDK", "2LDK", etc.
	re := regexp.MustCompile(`([0-9]?[SLDK]+)\b`)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// extractArea extracts area in square meters
func extractArea(text string) float64 {
	// Pattern: "25.5㎡" or "25.5m²"
	re := regexp.MustCompile(`([0-9]+\.?[0-9]*)[㎡m²]`)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		if val, err := strconv.ParseFloat(matches[1], 64); err == nil {
			// Validate: area should be reasonable (5-500 sqm for residential)
			if val >= 5.0 && val <= 500.0 {
				return val
			}
		}
	}
	return 0
}

// extractWalkTime extracts walking time to station in minutes
func extractWalkTime(text string) int {
	// Pattern: "徒歩5分" or "歩5分"
	re := regexp.MustCompile(`[徒歩]+([0-9]+)分`)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		if val, err := strconv.Atoi(matches[1]); err == nil {
			// Validate: walk time should be reasonable (1-60 minutes)
			if val >= 1 && val <= 60 {
				return val
			}
		}
	}
	return 0
}

// extractStation extracts station name
func extractStation(text string) string {
	// Pattern: "XX駅" before "徒歩"
	re := regexp.MustCompile(`([^\s]+駅)\s*[徒歩]`)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// extractAddress extracts address from the document
func extractAddress(doc *goquery.Document) string {
	// Try to find address in common patterns
	address := ""

	// Look for address in text
	doc.Find("*").Each(func(i int, s *goquery.Selection) {
		text := s.Text()
		if strings.Contains(text, "東京都") || strings.Contains(text, "大阪府") ||
		   strings.Contains(text, "神奈川県") || strings.Contains(text, "千葉県") ||
		   strings.Contains(text, "埼玉県") {
			// Extract just the address part
			re := regexp.MustCompile(`(東京都|大阪府|神奈川県|千葉県|埼玉県)[^\n]+`)
			matches := re.FindStringSubmatch(text)
			if len(matches) > 0 && len(address) == 0 {
				address = strings.TrimSpace(matches[0])
				// Safely truncate at rune boundary
				runes := []rune(address)
				if len(runes) > 50 {
					address = string(runes[:50])
				}
			}
		}
	})

	return address
}

// extractBuildingAge extracts building age in years
func extractBuildingAge(text string) int {
	// Pattern: "築5年" or "築年数5年"
	re := regexp.MustCompile(`築[年数]*([0-9]+)年`)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		if val, err := strconv.Atoi(matches[1]); err == nil {
			// Validate: building age should be reasonable (0-100 years)
			if val >= 0 && val <= 100 {
				return val
			}
		}
	}
	return 0
}

// extractFloor extracts floor number
func extractFloor(text string) int {
	// Pattern: "2階" or "2F"
	re := regexp.MustCompile(`([0-9]+)[階F]`)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		if val, err := strconv.Atoi(matches[1]); err == nil {
			// Validate: floor should be reasonable (0-100)
			if val >= 0 && val <= 100 {
				return val
			}
		}
	}
	return 0
}

// normalizeURL normalizes a URL by removing query strings and trailing slashes
func normalizeURL(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return rawURL // Return original if parsing fails
	}

	// Remove query strings and fragments
	parsedURL.RawQuery = ""
	parsedURL.Fragment = ""

	// Remove trailing slash from path
	if len(parsedURL.Path) > 1 && strings.HasSuffix(parsedURL.Path, "/") {
		parsedURL.Path = strings.TrimSuffix(parsedURL.Path, "/")
	}

	return parsedURL.String()
}

// extractCanonicalURL extracts canonical URL from HTML
func extractCanonicalURL(doc *goquery.Document) string {
	if canonicalURL, exists := doc.Find("link[rel='canonical']").Attr("href"); exists {
		return strings.TrimSpace(canonicalURL)
	}
	return ""
}

// extractExternalImageFromJSON extracts ExternalImageUrl from embedded JSON data
func extractExternalImageFromJSON(html string) string {
	// Look for ExternalImageUrl in the JSON data
	// Pattern: "ExternalImageUrl":"https://..."
	re := regexp.MustCompile(`"ExternalImageUrl":"([^"]+)"`)
	matches := re.FindStringSubmatch(html)
	if len(matches) > 1 {
		// Unescape JSON string (replace \/ with /)
		imageURL := strings.ReplaceAll(matches[1], `\/`, `/`)
		return imageURL
	}
	return ""
}

// verifyImageURL checks if an image URL is accessible (returns HTTP 200)
func (s *Scraper) verifyImageURL(imageURL string) bool {
	// Create HEAD request to check without downloading the image
	req, err := http.NewRequest("HEAD", imageURL, nil)
	if err != nil {
		log.Printf("[verifyImageURL] Error creating request for %s: %v", imageURL, err)
		return false
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

	// Use a shorter timeout for image verification
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[verifyImageURL] Error verifying image %s: %v", imageURL, err)
		return false
	}
	defer resp.Body.Close()

	// Accept 200 OK
	if resp.StatusCode != 200 {
		log.Printf("[verifyImageURL] Image verification failed for %s: status code %d", imageURL, resp.StatusCode)
		return false
	}

	log.Printf("[verifyImageURL] Image verified successfully: %s", imageURL)
	return true
}

// extractYahooPropertyID extracts Yahoo property ID from URL
// Example: https://realestate.yahoo.co.jp/rent/detail/000008250678c0a0c9accff94eab13c4c687966f0698
// Returns: 000008250678c0a0c9accff94eab13c4c687966f0698
func extractYahooPropertyID(detailURL string) (string, error) {
	// Split by /detail/
	parts := strings.Split(detailURL, "/detail/")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid Yahoo URL format (missing /detail/): %s", detailURL)
	}

	// Get the part after /detail/ and remove query params and trailing slash
	propertyID := strings.Split(parts[1], "?")[0]
	propertyID = strings.TrimSuffix(propertyID, "/")

	// Validate length (Yahoo property IDs are 48 hex characters)
	if len(propertyID) != 48 {
		return "", fmt.Errorf("unexpected property ID length %d (expected 48): %s", len(propertyID), propertyID)
	}

	// Validate that it's a hex string
	hexPattern := regexp.MustCompile(`^[0-9a-f]{48}$`)
	if !hexPattern.MatchString(propertyID) {
		return "", fmt.Errorf("invalid Yahoo property ID format (not hex): %s", propertyID)
	}

	return propertyID, nil
}
