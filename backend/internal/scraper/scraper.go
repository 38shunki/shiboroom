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
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
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

	for attempt := 0; attempt <= s.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: delay * 2^(attempt-1)
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * s.retryDelay
			log.Printf("Retry attempt %d/%d after %v", attempt, s.maxRetries, backoff)
			time.Sleep(backoff)
		}

		// Apply rate limiting
		s.rateLimit()

		resp, err = s.client.Do(req)
		if err == nil && resp.StatusCode == 200 {
			return resp, nil
		}

		// Log error for retry
		if err != nil {
			log.Printf("Request failed (attempt %d): %v", attempt+1, err)
		} else {
			log.Printf("Request failed (attempt %d): status code %d", attempt+1, resp.StatusCode)
			if resp.Body != nil {
				resp.Body.Close()
			}
		}

		// Don't retry on client errors (4xx)
		if resp != nil && resp.StatusCode >= 400 && resp.StatusCode < 500 {
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

	// Extract metadata
	property := &models.Property{
		DetailURL: normalizedURL,
		FetchedAt: time.Now(),
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

	// Generate ID from normalized URL hash
	hash := md5.Sum([]byte(normalizedURL))
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
