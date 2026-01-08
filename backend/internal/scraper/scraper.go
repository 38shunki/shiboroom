package scraper

import (
	"compress/gzip"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"real-estate-portal/internal/models"
	"real-estate-portal/internal/ratelimit"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
)

var (
	// Global rate limiter for Yahoo Real Estate list pages
	// Used for list page scraping (search results)
	yahooLimiter = ratelimit.NewYahooLimiter(
		1,                     // maxInFlight: 1 concurrent request (avoid burst)
		8000*time.Millisecond, // baseDelay: 8s base for list pages
		4000*time.Millisecond, // jitter: 0-4s (total: 8-12s)
	)

	// DetailLimiter is exported for use in API handlers (single detail page scraping)
	// Strictly limits detail pages to 8 per hour to avoid WAF detection
	// NOTE: This should ONLY be used for single /api/scrape requests, NOT for batch/list operations
	DetailLimiter = ratelimit.NewDetailLimiter(10) // 10 detail pages per hour max

	// Global circuit breaker to detect WAF blocks
	// Stricter early detection to avoid prolonged blocks
	circuitBreaker = NewCircuitBreaker(
		8,           // failureThreshold: 8 failures out of 20 requests (stricter)
		1*time.Hour, // resetTimeout: wait 1 hour before retry
	)
)

type Scraper struct {
	client                *http.Client
	maxRetries            int
	retryDelay            time.Duration
	requestDelay          time.Duration
	lastRequestTime       time.Time
	lastHomepageVisit     time.Time
	homepageVisitInterval time.Duration
	lastStations          []StationAccess // Stores stations from the last scrape
	lastImages            []string        // Stores image URLs from the last scrape
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
	// Create cookie jar for session management
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Printf("Warning: Failed to create cookie jar: %v", err)
		jar = nil
	}

	return &Scraper{
		client: &http.Client{
			Timeout: config.Timeout,
			Jar:     jar,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Follow redirects while maintaining cookies
				return nil
			},
		},
		maxRetries:            config.MaxRetries,
		retryDelay:            config.RetryDelay,
		requestDelay:          config.RequestDelay,
		homepageVisitInterval: 30 * time.Minute, // Visit homepage every 30 minutes to maintain session
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

// visitHomepageIfNeeded visits the Yahoo Real Estate homepage to establish a session
// This helps avoid bot detection by simulating a real user browsing flow
func (s *Scraper) visitHomepageIfNeeded() error {
	// Check if we need to visit homepage
	if time.Since(s.lastHomepageVisit) < s.homepageVisitInterval {
		return nil // Recent visit, no need to visit again
	}

	log.Printf("[Homepage] Visiting Yahoo Real Estate homepage to establish session")

	req, err := http.NewRequest("GET", "https://realestate.yahoo.co.jp/", nil)
	if err != nil {
		return err
	}

	applyBrowserHeaders(req, "")

	resp, err := s.client.Do(req)
	if err != nil {
		log.Printf("[Homepage] Error visiting homepage: %v", err)
		return err
	}
	defer resp.Body.Close()

	s.lastHomepageVisit = time.Now()
	log.Printf("[Homepage] Successfully visited homepage (Status: %d), cookies saved", resp.StatusCode)

	// Small delay after homepage visit to appear more natural
	time.Sleep(time.Duration(2+rand.Intn(3)) * time.Second)

	return nil
}

// applyBrowserHeaders sets browser-like headers to avoid bot detection
func applyBrowserHeaders(req *http.Request, referer string) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Accept-Language", "ja-JP,ja;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")
	req.Header.Set("DNT", "1")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("sec-ch-ua", `"Not A(Brand";v="99", "Google Chrome";v="122", "Chromium";v="122"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"Windows"`)

	if referer != "" {
		req.Header.Set("Referer", referer)
		req.Header.Set("Sec-Fetch-Site", "same-origin")
	}
}

// isWAFBlock checks if a response indicates a WAF block
func isWAFBlock(resp *http.Response) bool {
	if resp.StatusCode != 500 {
		return false
	}

	// Read body to check for WAF indicators
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}

	// Replace body so it can be read again if needed
	resp.Body = io.NopCloser(strings.NewReader(string(body)))

	bodyStr := string(body)
	// Check for Yahoo WAF block message
	if strings.Contains(bodyStr, "ご覧になろうとしているページは現在表示できません") {
		log.Printf("[WAF] Detected Yahoo WAF block page")
		return true
	}

	return false
}

// sleepHumanDetailPace simulates human browsing behavior with natural delays
func sleepHumanDetailPace() {
	// 80% normal browsing (45-120 seconds)
	// 20% deep reading (180-420 seconds = 3-7 minutes)
	p := rand.Float64()
	var duration time.Duration

	if p < 0.8 {
		// Normal: 45-120 seconds
		duration = time.Duration(45+rand.Intn(76)) * time.Second
	} else {
		// Deep reading: 180-420 seconds
		duration = time.Duration(180+rand.Intn(241)) * time.Second
	}

	log.Printf("[Human Pace] Sleeping for %v to simulate human browsing", duration)
	time.Sleep(duration)
}

// sleepHumanListPace simulates human browsing behavior for list pages
func sleepHumanListPace() {
	// List pages: 6-15 seconds
	duration := time.Duration(6+rand.Intn(10)) * time.Second
	log.Printf("[Human Pace] Sleeping for %v for list page browsing", duration)
	time.Sleep(duration)
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

			// Check for WAF block - immediate failure, no retry
			if isWAFBlock(resp) {
				circuitBreaker.RecordFailure(resp.StatusCode)
				if resp.Body != nil {
					resp.Body.Close()
				}
				return nil, fmt.Errorf("WAF block detected: immediate retreat required")
			}

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
			// 404: Property not found / delisted (permanent failure, not WAF)
			if resp.StatusCode == 404 {
				log.Printf("404 Not Found (property likely delisted): not retrying")
			}
			break
		}
	}

	if err != nil {
		return nil, fmt.Errorf("request failed after %d retries: %w", s.maxRetries, err)
	}
	// Include status code in error for caller to distinguish 404 vs WAF
	if resp != nil && resp.StatusCode == 404 {
		return nil, fmt.Errorf("permanent_fail: status code 404 (property not found or delisted)")
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

	// Apply browser-like headers (no referer for list page)
	applyBrowserHeaders(req, "")

	resp, err := s.doRequestWithRetry(req)
	if err != nil {
		log.Printf("[ScrapeListPage] Error fetching list page %s: %v", listURL, err)
		return nil, fmt.Errorf("failed to fetch list page: %w", err)
	}
	defer resp.Body.Close()

	// Handle gzip decompression if needed
	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			log.Printf("[ScrapeListPage] Error creating gzip reader: %v", err)
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzipReader.Close()
		reader = gzipReader
	}

	// Parse HTML (goquery will read body completely, maintaining connection stability)
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		log.Printf("[ScrapeListPage] Error parsing HTML from %s: %v", listURL, err)
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var propertyURLs []string
	seenURLs := make(map[string]bool)

	// Find all property checkboxes (Yahoo changed HTML structure - property IDs are now in checkbox values)
	// Property IDs are 40-character hex strings (NOT 48) in input._propertyCheckbox value attributes
	// Note: As of Dec 2024, Yahoo uses these formats:
	//   - "_0000" prefix + 40-char ID (45 chars total)
	//   - "0000" prefix + 40-char ID (44 chars total)
	//   - 40-char ID (no prefix, 40 chars total)
	doc.Find("input._propertyCheckbox").Each(func(i int, s *goquery.Selection) {
		value, exists := s.Attr("value")

		if !exists {
			return
		}
		if len(value) < 40 {
			return
		}

		// Use the value as-is (Yahoo changed their ID format in late 2024)
		// Property IDs can be 40-48 characters with various prefixes
		// The href in the HTML uses the full value, so we should too
		propertyID := value

		// Skip if empty
		if propertyID == "" {
			return
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
	})

	log.Printf("[ScrapeListPage] Found %d unique property URLs from %s", len(propertyURLs), listURL)
	return propertyURLs, nil
}

// fetchHTMLWithHeadlessBrowser uses Chrome headless browser to fetch HTML
// This bypasses most anti-bot detection by executing JavaScript
func (s *Scraper) fetchHTMLWithHeadlessBrowser(url string) (string, error) {
	log.Printf("[HeadlessBrowser] Fetching %s with Chrome", url)

	// Chrome execution options for systemd compatibility
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.ExecPath("/usr/bin/google-chrome"), // Use Google Chrome
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true), // Required for systemd/Docker
		chromedp.Flag("disable-dev-shm-usage", true), // Prevents /dev/shm issues
		chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.Flag("disable-software-rasterizer", true),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"),
	)

	// Create allocator context with Chrome options
	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocCancel()

	// Create browser context
	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Set a timeout for the entire operation (30 seconds)
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var htmlContent string
	err := chromedp.Run(ctx,
		// Navigate to the URL
		chromedp.Navigate(url),
		// Wait for the page to load (wait for body element)
		chromedp.WaitVisible(`body`, chromedp.ByQuery),
		// Wait a bit more for JavaScript to execute
		chromedp.Sleep(3*time.Second),
		// Get the rendered HTML
		chromedp.OuterHTML(`html`, &htmlContent, chromedp.ByQuery),
	)

	if err != nil {
		log.Printf("[HeadlessBrowser] ERROR fetching %s: %v", url, err)
		return "", fmt.Errorf("chromedp error: %w", err)
	}

	// Log HTML size and preview
	htmlSize := len(htmlContent)
	previewLen := 500
	if htmlSize < previewLen {
		previewLen = htmlSize
	}
	log.Printf("[HeadlessBrowser] Successfully fetched HTML (%d bytes)", htmlSize)
	log.Printf("[HeadlessBrowser] HTML preview (first %d chars): %s", previewLen, htmlContent[:previewLen])

	return htmlContent, nil
}

// ScrapeProperty scrapes a property detail page
func (s *Scraper) ScrapeProperty(inputURL string) (*models.Property, error) {
	return s.ScrapePropertyWithReferer(inputURL, "")
}

// ScrapePropertyWithReferer scrapes a property detail page with optional referer
// NOTE: Rate limiting (DetailLimiter) should be applied by the caller, not here.
// This function only applies human-like delay to avoid detection.
func (s *Scraper) ScrapePropertyWithReferer(inputURL string, referer string) (*models.Property, error) {
	// Normalize URL (remove query strings, trailing slash)
	normalizedURL := normalizeURL(inputURL)
	log.Printf("[ScrapeProperty] Starting scrape of property: %s (normalized: %s, referer: %s)", inputURL, normalizedURL, referer)

	// Visit homepage if needed to establish/maintain session
	if err := s.visitHomepageIfNeeded(); err != nil {
		log.Printf("[ScrapeProperty] Warning: Failed to visit homepage: %v", err)
		// Continue anyway, as this is not a critical error
	}

	// Sleep to simulate human browsing behavior (45-120s, sometimes 3-7 minutes)
	// NOTE: DetailLimiter.Acquire() should be called by the caller before this function
	sleepHumanDetailPace()

	// Fetch the page using headless browser
	htmlContent, err := s.fetchHTMLWithHeadlessBrowser(normalizedURL)
	if err != nil {
		log.Printf("[ScrapeProperty] Error fetching URL with headless browser %s: %v", normalizedURL, err)
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}

	// Parse HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
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

	// Extract title with priority: og:title -> twitter:title -> title tag -> h1
	// Add detailed logging to diagnose extraction issues
	ogTitle, ogExists := doc.Find("meta[property='og:title']").Attr("content")
	twitterTitle, twitterExists := doc.Find("meta[name='twitter:title']").Attr("content")
	titleTag := strings.TrimSpace(doc.Find("title").Text())
	h1Tag := strings.TrimSpace(doc.Find("h1").First().Text())

	log.Printf("[ScrapeProperty] Title extraction debug for %s:", normalizedURL)
	log.Printf("  - og:title exists: %v, value: %q", ogExists, ogTitle)
	log.Printf("  - twitter:title exists: %v, value: %q", twitterExists, twitterTitle)
	log.Printf("  - <title> tag: %q", titleTag)
	log.Printf("  - <h1> tag: %q", h1Tag)

	if ogExists && strings.TrimSpace(ogTitle) != "" {
		property.Title = strings.TrimSpace(ogTitle)
	} else if twitterExists && strings.TrimSpace(twitterTitle) != "" {
		property.Title = strings.TrimSpace(twitterTitle)
	} else if titleTag != "" {
		property.Title = titleTag
	} else if h1Tag != "" {
		property.Title = h1Tag
	} else {
		property.Title = "No Title"
		log.Printf("[ScrapeProperty] Warning: Could not extract title from %s", normalizedURL)
	}

	// Clean up title: remove "Yahoo不動産" and related text
	property.Title = cleanTitle(property.Title)

	// Fallback check after cleanup
	if property.Title == "" {
		property.Title = "No Title"
		log.Printf("[ScrapeProperty] Warning: Title became empty after cleanup for %s", normalizedURL)
		// Log first 500 chars of HTML to diagnose
		if htmlContent, err := doc.Html(); err == nil && len(htmlContent) > 0 {
			previewLen := 500
			if len(htmlContent) < previewLen {
				previewLen = len(htmlContent)
			}
			log.Printf("[ScrapeProperty] HTML preview (first %d chars): %s", previewLen, htmlContent[:previewLen])
		}
	}

	// Extract all image URLs from the page
	pageHTML, _ := doc.Html()
	allImageURLs := extractAllImageURLsFromJSON(pageHTML)

	// Store all image URLs for later retrieval
	s.lastImages = allImageURLs

	// Set the first image as the primary image for backward compatibility
	if len(allImageURLs) > 0 {
		property.ImageURL = allImageURLs[0]
		log.Printf("[ScrapeProperty] Set primary image from %d total images", len(allImageURLs))
	} else {
		// Fallback to og:image if no images found in JSON
		if imageURL, exists := doc.Find("meta[property='og:image']").Attr("content"); exists {
			imageURL = strings.TrimSpace(imageURL)
			if s.verifyImageURL(imageURL) {
				property.ImageURL = imageURL
				s.lastImages = []string{imageURL}
				log.Printf("[ScrapeProperty] Using og:image as fallback")
			}
		}
	}

	// Extract additional details from the page
	s.extractDetailFields(doc, property)

	// Extract stations (new: for property_stations table)
	// Apply backward compatibility by copying sort_order=1 to legacy fields
	stations := extractStations(doc)
	applyStationCompatibility(property, stations)
	// Store stations in scraper for retrieval by API handler
	s.lastStations = stations
	// Note: The actual saving to property_stations table happens in the API handler
	// via gormDB.SavePropertyWithStations()

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

	log.Printf("[ScrapeProperty] Successfully scraped property %s (ID: %s, Title: %s, Stations: %d)", normalizedURL, property.ID, property.Title, len(stations))
	return property, nil
}

// extractPropertyDataFromHTML extracts property data directly from __SERVER_SIDE_CONTEXT__ using regex
// This is more reliable than trying to parse JavaScript object literals as JSON
func extractPropertyDataFromHTML(htmlString string) map[string]interface{} {
	result := make(map[string]interface{})

	// Find __SERVER_SIDE_CONTEXT__ section
	ctxIdx := strings.Index(htmlString, "__SERVER_SIDE_CONTEXT__")
	if ctxIdx == -1 {
		log.Printf("[extractPropertyDataFromHTML] __SERVER_SIDE_CONTEXT__ not found")
		return result
	}

	// Get a reasonable chunk after __SERVER_SIDE_CONTEXT__ (next 500KB should be enough)
	endIdx := ctxIdx + 500000
	if endIdx > len(htmlString) {
		endIdx = len(htmlString)
	}
	contextSection := htmlString[ctxIdx:endIdx]

	// Extract Price (rent)
	if re := regexp.MustCompile(`"Price"\s*:\s*(\d+)`); re != nil {
		if matches := re.FindStringSubmatch(contextSection); len(matches) > 1 {
			if price, err := strconv.Atoi(matches[1]); err == nil {
				result["Price"] = price
				log.Printf("[extractPropertyDataFromHTML] Found Price: %d", price)
			}
		}
	}

	// Extract BuildingName
	if re := regexp.MustCompile(`"BuildingName"\s*:\s*"([^"]+)"`); re != nil {
		if matches := re.FindStringSubmatch(contextSection); len(matches) > 1 {
			result["BuildingName"] = matches[1]
			log.Printf("[extractPropertyDataFromHTML] Found BuildingName: %s", matches[1])
		}
	}

	// Extract MonopolyArea
	if re := regexp.MustCompile(`"MonopolyArea"\s*:\s*(\d+)`); re != nil {
		if matches := re.FindStringSubmatch(contextSection); len(matches) > 1 {
			if area, err := strconv.Atoi(matches[1]); err == nil {
				result["MonopolyArea"] = area
				log.Printf("[extractPropertyDataFromHTML] Found MonopolyArea: %d", area)
			}
		}
	}

	// Extract MinutesFromStation
	if re := regexp.MustCompile(`"MinutesFromStation"\s*:\s*(\d+)`); re != nil {
		if matches := re.FindStringSubmatch(contextSection); len(matches) > 1 {
			if minutes, err := strconv.Atoi(matches[1]); err == nil {
				result["MinutesFromStation"] = minutes
				log.Printf("[extractPropertyDataFromHTML] Found MinutesFromStation: %d", minutes)
			}
		}
	}

	// Extract FloorNum
	if re := regexp.MustCompile(`"FloorNum"\s*:\s*"?(\d+)"?`); re != nil {
		if matches := re.FindStringSubmatch(contextSection); len(matches) > 1 {
			if floor, err := strconv.Atoi(matches[1]); err == nil {
				result["FloorNum"] = floor
				log.Printf("[extractPropertyDataFromHTML] Found FloorNum: %d", floor)
			}
		}
	}

	// Extract AddressName
	if re := regexp.MustCompile(`"AddressName"\s*:\s*"([^"]+)"`); re != nil {
		if matches := re.FindStringSubmatch(contextSection); len(matches) > 1 {
			result["AddressName"] = matches[1]
			log.Printf("[extractPropertyDataFromHTML] Found AddressName: %s", matches[1])
		}
	}

	// Extract StationName
	if re := regexp.MustCompile(`"StationName"\s*:\s*"([^"]+)"`); re != nil {
		if matches := re.FindStringSubmatch(contextSection); len(matches) > 1 {
			result["StationName"] = matches[1]
			log.Printf("[extractPropertyDataFromHTML] Found StationName: %s", matches[1])
		}
	}

	// Extract YearsOld (building age)
	if re := regexp.MustCompile(`"YearsOld"\s*:\s*(\d+)`); re != nil {
		if matches := re.FindStringSubmatch(contextSection); len(matches) > 1 {
			if yearsOld, err := strconv.Atoi(matches[1]); err == nil {
				result["YearsOld"] = yearsOld
				log.Printf("[extractPropertyDataFromHTML] Found YearsOld: %d", yearsOld)
			}
		}
	}

	// Extract Direction (orientation)
	if re := regexp.MustCompile(`"Direction"\s*:\s*"([^"]+)"`); re != nil {
		if matches := re.FindStringSubmatch(contextSection); len(matches) > 1 {
			result["Direction"] = matches[1]
			log.Printf("[extractPropertyDataFromHTML] Found Direction: %s", matches[1])
		}
	}

	// Extract StructureName (building structure)
	if re := regexp.MustCompile(`"StructureName"\s*:\s*"([^"]+)"`); re != nil {
		if matches := re.FindStringSubmatch(contextSection); len(matches) > 1 {
			result["StructureName"] = matches[1]
			log.Printf("[extractPropertyDataFromHTML] Found StructureName: %s", matches[1])
		}
	}

	// Extract RoomLayoutBreakdown (detailed floor plan)
	if re := regexp.MustCompile(`"RoomLayoutBreakdown"\s*:\s*"([^"]+)"`); re != nil {
		if matches := re.FindStringSubmatch(contextSection); len(matches) > 1 {
			result["RoomLayoutBreakdown"] = matches[1]
			log.Printf("[extractPropertyDataFromHTML] Found RoomLayoutBreakdown: %s", matches[1])
		}
	}

	// Extract KindName (building type: マンション/アパート/一戸建て)
	if re := regexp.MustCompile(`"KindName"\s*:\s*"([^"]+)"`); re != nil {
		if matches := re.FindStringSubmatch(contextSection); len(matches) > 1 {
			result["KindName"] = matches[1]
			log.Printf("[extractPropertyDataFromHTML] Found KindName: %s", matches[1])
		}
	}

	// Extract Facilities (こだわり条件 - array of codes)
	if re := regexp.MustCompile(`"Facilities"\s*:\s*(\[[^\]]*\])`); re != nil {
		if matches := re.FindStringSubmatch(contextSection); len(matches) > 1 {
			result["Facilities"] = matches[1]
			log.Printf("[extractPropertyDataFromHTML] Found Facilities: %s", matches[1])
		}
	}

	// Extract Pickouts (特徴 - array of feature codes)
	if re := regexp.MustCompile(`"Pickouts"\s*:\s*(\[[^\]]*\])`); re != nil {
		if matches := re.FindStringSubmatch(contextSection); len(matches) > 1 {
			result["Pickouts"] = matches[1]
			log.Printf("[extractPropertyDataFromHTML] Found Pickouts: %s", matches[1])
		}
	}

	// Extract BuildingName (建物名)
	if re := regexp.MustCompile(`"BuildingName"\s*:\s*"([^"]+)"`); re != nil {
		if matches := re.FindStringSubmatch(contextSection); len(matches) > 1 {
			result["BuildingName"] = matches[1]
			log.Printf("[extractPropertyDataFromHTML] Found BuildingName: %s", matches[1])
		}
	}

	// Extract FloorNameLabel (階数情報)
	if re := regexp.MustCompile(`"FloorNameLabel"\s*:\s*"([^"]+)"`); re != nil {
		if matches := re.FindStringSubmatch(contextSection); len(matches) > 1 {
			result["FloorNameLabel"] = matches[1]
			log.Printf("[extractPropertyDataFromHTML] Found FloorNameLabel: %s", matches[1])
		}
	}

	// Extract FloorNum (階数)
	if re := regexp.MustCompile(`"FloorNum"\s*:\s*(\d+)`); re != nil {
		if matches := re.FindStringSubmatch(contextSection); len(matches) > 1 {
			if floorNum, err := strconv.Atoi(matches[1]); err == nil {
				result["FloorNum"] = floorNum
				log.Printf("[extractPropertyDataFromHTML] Found FloorNum: %d", floorNum)
			}
		}
	}

	// Extract ParkingAreaLabel (駐車場)
	if re := regexp.MustCompile(`"ParkingAreaLabel"\s*:\s*"([^"]+)"`); re != nil {
		if matches := re.FindStringSubmatch(contextSection); len(matches) > 1 {
			result["ParkingAreaLabel"] = matches[1]
			log.Printf("[extractPropertyDataFromHTML] Found ParkingAreaLabel: %s", matches[1])
		}
	}

	// Extract ContractPeriod (契約期間)
	if re := regexp.MustCompile(`"ContractPeriod"\s*:\s*"([^"]+)"`); re != nil {
		if matches := re.FindStringSubmatch(contextSection); len(matches) > 1 {
			result["ContractPeriod"] = matches[1]
			log.Printf("[extractPropertyDataFromHTML] Found ContractPeriod: %s", matches[1])
		}
	}

	// Extract Insurance (保険)
	if re := regexp.MustCompile(`"Insurance"\s*:\s*"([^"]+)"`); re != nil {
		if matches := re.FindStringSubmatch(contextSection); len(matches) > 1 {
			result["Insurance"] = matches[1]
			log.Printf("[extractPropertyDataFromHTML] Found Insurance: %s", matches[1])
		}
	}

	// Extract RoomLayoutImageUrl (間取り図URL)
	if re := regexp.MustCompile(`"RoomLayoutImageUrl"\s*:\s*"([^"]+)"`); re != nil {
		if matches := re.FindStringSubmatch(contextSection); len(matches) > 1 {
			// Unescape JSON string (remove \/ escapes)
			unescapedURL := strings.ReplaceAll(matches[1], `\/`, `/`)
			result["RoomLayoutImageUrl"] = unescapedURL
			log.Printf("[extractPropertyDataFromHTML] Found RoomLayoutImageUrl: %s", unescapedURL)
		}
	}

	log.Printf("[extractPropertyDataFromHTML] Extracted %d fields", len(result))
	return result
}

// extractServerSideContextJSON extracts __SERVER_SIDE_CONTEXT__ JSON from HTML
func extractServerSideContextJSON(htmlString string) (map[string]interface{}, error) {
	// Find script tags containing __SERVER_SIDE_CONTEXT__
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlString))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var scriptText string
	doc.Find("script").EachWithBreak(func(_ int, s *goquery.Selection) bool {
		text := s.Text()
		if strings.Contains(text, "__SERVER_SIDE_CONTEXT__") {
			scriptText = text
			return false // Stop iteration
		}
		return true
	})

	if scriptText == "" {
		return nil, fmt.Errorf("__SERVER_SIDE_CONTEXT__ script not found")
	}

	// Find the assignment: __SERVER_SIDE_CONTEXT__ = ...
	assignmentIdx := strings.Index(scriptText, "__SERVER_SIDE_CONTEXT__")
	if assignmentIdx == -1 {
		return nil, fmt.Errorf("__SERVER_SIDE_CONTEXT__ not found in script")
	}

	// Find the = sign after __SERVER_SIDE_CONTEXT__
	afterVar := scriptText[assignmentIdx+len("__SERVER_SIDE_CONTEXT__"):]
	eqIdx := strings.Index(afterVar, "=")
	if eqIdx == -1 {
		return nil, fmt.Errorf("assignment operator not found")
	}

	// Get everything after the =
	afterEq := strings.TrimSpace(afterVar[eqIdx+1:])

	// Try to extract JSON
	var jsonStr string

	// Pattern 1: JSON object starting with {
	if strings.HasPrefix(afterEq, "{") {
		// Find the matching closing brace using a brace counter
		braceCount := 0
		inString := false
		escaped := false
		endIdx := -1

		for i, ch := range afterEq {
			if escaped {
				escaped = false
				continue
			}

			if ch == '\\' {
				escaped = true
				continue
			}

			if ch == '"' {
				inString = !inString
				continue
			}

			if !inString {
				if ch == '{' {
					braceCount++
				} else if ch == '}' {
					braceCount--
					if braceCount == 0 {
						endIdx = i + 1
						break
					}
				}
			}
		}

		if endIdx > 0 {
			jsonStr = afterEq[:endIdx]
		} else {
			return nil, fmt.Errorf("could not find matching closing brace")
		}
	} else if strings.HasPrefix(afterEq, "\"") {
		// Pattern 2: JSON string "..."
		// Find the closing quote
		endIdx := -1
		escaped := false
		for i := 1; i < len(afterEq); i++ {
			if escaped {
				escaped = false
				continue
			}
			if afterEq[i] == '\\' {
				escaped = true
				continue
			}
			if afterEq[i] == '"' {
				endIdx = i + 1
				break
			}
		}

		if endIdx > 0 {
			// Unquote the string
			quoted := afterEq[:endIdx]
			unquoted, err := strconv.Unquote(quoted)
			if err != nil {
				return nil, fmt.Errorf("failed to unquote JSON string: %w", err)
			}
			jsonStr = unquoted
		} else {
			return nil, fmt.Errorf("could not find closing quote")
		}
	} else {
		return nil, fmt.Errorf("unexpected JSON format (does not start with { or \")")
	}

	// Unescape HTML entities
	jsonStr = html.UnescapeString(jsonStr)

	// Yahoo uses JavaScript object literal format (unquoted keys), not standard JSON
	// Convert to standard JSON by adding quotes around keys
	jsonStr = convertJSObjectToJSON(jsonStr)

	// Parse JSON
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		// Log first 500 chars of JSON for debugging
		preview := jsonStr
		if len(preview) > 500 {
			preview = preview[:500] + "..."
		}
		log.Printf("[extractServerSideContextJSON] Failed to parse JSON: %v (preview: %s)", err, preview)
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	log.Printf("[extractServerSideContextJSON] Successfully parsed JSON, keys: %d", len(result))
	return result, nil
}

// convertJSObjectToJSON converts JavaScript object literal to valid JSON
// Changes: {foo: "bar"} -> {"foo": "bar"}
func convertJSObjectToJSON(jsObj string) string {
	// Use regexp to add quotes around unquoted keys
	// Pattern: match word characters followed by colon (not already quoted)
	re := regexp.MustCompile(`([,{]\s*)([a-zA-Z_][a-zA-Z0-9_]*)(\s*:)`)
	return re.ReplaceAllString(jsObj, `$1"$2"$3`)
}

// getNestedValue retrieves a nested value from a map using dot notation (e.g., "page.property.Price")
func getNestedValue(m map[string]interface{}, path string) (interface{}, bool) {
	var current interface{} = m
	for _, key := range strings.Split(path, ".") {
		obj, ok := current.(map[string]interface{})
		if !ok {
			return nil, false
		}
		current, ok = obj[key]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

// getInt extracts an integer value from nested map
func getInt(m map[string]interface{}, path string) (int, bool) {
	v, ok := getNestedValue(m, path)
	if !ok || v == nil {
		return 0, false
	}
	switch t := v.(type) {
	case float64:
		return int(t), true
	case string:
		i, err := strconv.Atoi(strings.TrimSpace(t))
		return i, err == nil
	default:
		return 0, false
	}
}

// getFloat extracts a float value from nested map
func getFloat(m map[string]interface{}, path string) (float64, bool) {
	v, ok := getNestedValue(m, path)
	if !ok || v == nil {
		return 0, false
	}
	switch t := v.(type) {
	case float64:
		return t, true
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(t), 64)
		return f, err == nil
	default:
		return 0, false
	}
}

// getString extracts a string value from nested map
func getString(m map[string]interface{}, path string) (string, bool) {
	v, ok := getNestedValue(m, path)
	if !ok || v == nil {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// extractDetailFields extracts detailed property information from the DOM
// First tries to extract from __SERVER_SIDE_CONTEXT__ using regex, then falls back to DOM scraping
func (s *Scraper) extractDetailFields(doc *goquery.Document, property *models.Property) {
	// Try to extract from __SERVER_SIDE_CONTEXT__ using direct regex first (most reliable)
	htmlString, _ := doc.Html()
	contextData := extractPropertyDataFromHTML(htmlString)

	if len(contextData) > 0 {
		log.Printf("[extractDetailFields] Found __SERVER_SIDE_CONTEXT__ data, extracting %d fields", len(contextData))
		s.extractFromContextData(contextData, property)

		// Also extract facilities from HTML labels (人気の特徴・設備 + category lines)
		// This handles properties that use Japanese labels instead of internal codes
		popularLabels := extractPopularFeatureLabels(doc)
		categoryLabels := extractCategoryFacilityLabels(doc)
		allLabels := append(popularLabels, categoryLabels...)

		if len(allLabels) > 0 {
			log.Printf("[extractDetailFields] id=%s Extracted %d facility labels from HTML", property.ID, len(allLabels))
			labelKeys := normalizeFacilitiesFromLabels(allLabels)

			// Combine code-based and label-based facilities (union operation)
			var existingKeys []string
			if property.Facilities != "" {
				json.Unmarshal([]byte(property.Facilities), &existingKeys)
			}

			// Merge and deduplicate
			allKeys := append(existingKeys, labelKeys...)
			uniqueKeys := make(map[string]bool)
			for _, key := range allKeys {
				if key != "" {
					uniqueKeys[key] = true
				}
			}

			// Convert back to sorted array
			finalKeys := make([]string, 0, len(uniqueKeys))
			for key := range uniqueKeys {
				finalKeys = append(finalKeys, key)
			}
			sort.Strings(finalKeys)

			if len(finalKeys) > 0 {
				result, _ := json.Marshal(finalKeys)
				property.Facilities = string(result)
				log.Printf("[extractDetailFields] id=%s Combined facilities: %s", property.ID, property.Facilities)
			}
		}

		return
	}

	log.Printf("[extractDetailFields] __SERVER_SIDE_CONTEXT__ not found, falling back to DOM extraction")

	// Fallback: Extract from the page text (best effort)
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

// decodeUnicodeEscape decodes Unicode escape sequences like \u6771\u4EAC
func decodeUnicodeEscape(s string) string {
	// Use strconv.Unquote to decode Unicode escapes
	// We need to wrap the string in quotes for Unquote to work
	quoted := `"` + s + `"`
	unquoted, err := strconv.Unquote(quoted)
	if err != nil {
		// If decoding fails, return original string
		return s
	}
	return unquoted
}

// extractFromContextData extracts property data from __SERVER_SIDE_CONTEXT__ data map
func (s *Scraper) extractFromContextData(contextData map[string]interface{}, property *models.Property) {
	propertyID := property.SourcePropertyID
	log.Printf("[extractFromContextData] id=%s Starting extraction from %d fields", propertyID, len(contextData))

	// Extract rent (Price)
	if price, ok := contextData["Price"].(int); ok && price > 0 {
		property.Rent = &price
		log.Printf("[extractFromContextData] id=%s Rent: %d", propertyID, price)
	}

	// Extract building name
	if buildingName, ok := contextData["BuildingName"].(string); ok && buildingName != "" {
		property.BuildingName = decodeUnicodeEscape(buildingName)
		// Use building name as title if title is empty or "No Title"
		if property.Title == "" || property.Title == "No Title" {
			property.Title = property.BuildingName
		}
		log.Printf("[extractFromContextData] id=%s BuildingName: %s", propertyID, property.BuildingName)
	}

	// Extract floor number
	if floorNum, ok := contextData["FloorNum"].(int); ok {
		property.Floor = &floorNum
	}

	// Extract area (MonopolyArea is in units of 0.01 sqm, need to divide by 100)
	if monopolyArea, ok := contextData["MonopolyArea"].(int); ok && monopolyArea > 0 {
		area := float64(monopolyArea) / 100.0
		property.Area = &area
		log.Printf("[extractFromContextData] id=%s Area: %.2f sqm", propertyID, area)
	}

	// Extract walk time
	if walkTime, ok := contextData["MinutesFromStation"].(int); ok && walkTime > 0 {
		property.WalkTime = &walkTime
	}

	// Extract address (decode Unicode escapes)
	if address, ok := contextData["AddressName"].(string); ok && address != "" {
		property.Address = decodeUnicodeEscape(address)
	}

	// Extract station name (decode Unicode escapes)
	if stationName, ok := contextData["StationName"].(string); ok && stationName != "" {
		property.Station = decodeUnicodeEscape(stationName)
		log.Printf("[extractFromContextData] id=%s Station: %s", propertyID, property.Station)
	}

	// Extract building age (YearsOld)
	if yearsOld, ok := contextData["YearsOld"].(int); ok && yearsOld >= 0 {
		property.BuildingAge = &yearsOld
		log.Printf("[extractFromContextData] id=%s BuildingAge: %d years", propertyID, yearsOld)
	}

	// Extract floor plan (RoomLayoutBreakdown, decode Unicode escapes)
	if roomLayout, ok := contextData["RoomLayoutBreakdown"].(string); ok && roomLayout != "" {
		decodedLayout := decodeUnicodeEscape(roomLayout)
		property.FloorPlanDetails = decodedLayout
		// If FloorPlan is empty, use RoomLayoutBreakdown
		if property.FloorPlan == "" {
			// Normalize the floor plan to standard codes (ワンルーム → 1R, etc.)
			property.FloorPlan = normalizeFloorPlan(decodedLayout)
			log.Printf("[extractFromContextData] id=%s FloorPlan: %s (normalized from: %s)", propertyID, property.FloorPlan, decodedLayout)
		}
		log.Printf("[extractFromContextData] id=%s FloorPlanDetails: %s", propertyID, property.FloorPlanDetails)
	}

	// Extract structure first (StructureName: 鉄筋コンクリート/軽量鉄骨等) - needed for building type classification
	var structureName string
	if structure, ok := contextData["StructureName"].(string); ok && structure != "" {
		structureName = decodeUnicodeEscape(structure)
		property.Structure = structureName
		log.Printf("[extractFromContextData] id=%s Structure: %s", propertyID, property.Structure)
	}

	// Extract building type (KindName: マンション/アパート/一戸建て)
	// Use structure for classification (RC = mansion, wooden = apartment)
	if kindName, ok := contextData["KindName"].(string); ok && kindName != "" {
		decodedType := decodeUnicodeEscape(kindName)
		// Normalize building type using both label and structure (structure takes priority)
		property.BuildingType = normalizeBuildingType(decodedType, structureName)
		log.Printf("[extractFromContextData] id=%s BuildingType: %s (normalized from: %s, structure: %s)",
			propertyID, property.BuildingType, decodedType, structureName)
	}

	// Extract facilities (こだわり条件 - stored as JSON string, normalize Yahoo codes to English keys)
	if facilities, ok := contextData["Facilities"].(string); ok && facilities != "" {
		// Normalize Yahoo facility codes to English keys for filtering
		normalizedFacilities := normalizeFacilities(facilities)
		property.Facilities = normalizedFacilities
		log.Printf("[extractFromContextData] id=%s Facilities: %s (normalized from: %s)", propertyID, normalizedFacilities, facilities)
	}

	// Extract features/pickouts (特徴 - stored as JSON string)
	if pickouts, ok := contextData["Pickouts"].(string); ok && pickouts != "" {
		property.Features = pickouts
		log.Printf("[extractFromContextData] id=%s Features: %s", propertyID, pickouts)
	}

	// Extract direction/orientation (方位)
	if direction, ok := contextData["Direction"].(string); ok && direction != "" {
		property.Direction = decodeUnicodeEscape(direction)
		log.Printf("[extractFromContextData] id=%s Direction: %s", propertyID, property.Direction)
	}

	// Extract floor label (階数情報)
	if floorLabel, ok := contextData["FloorNameLabel"].(string); ok && floorLabel != "" {
		property.FloorLabel = decodeUnicodeEscape(floorLabel)
		log.Printf("[extractFromContextData] id=%s FloorLabel: %s", propertyID, property.FloorLabel)
	}

	// Extract parking information
	if parking, ok := contextData["ParkingAreaLabel"].(string); ok && parking != "" {
		property.Parking = decodeUnicodeEscape(parking)
		log.Printf("[extractFromContextData] id=%s Parking: %s", propertyID, property.Parking)
	}

	// Extract contract period (契約期間)
	if contractPeriod, ok := contextData["ContractPeriod"].(string); ok && contractPeriod != "" {
		property.ContractPeriod = decodeUnicodeEscape(contractPeriod)
		log.Printf("[extractFromContextData] id=%s ContractPeriod: %s", propertyID, property.ContractPeriod)
	}

	// Extract insurance information
	if insurance, ok := contextData["Insurance"].(string); ok && insurance != "" {
		property.Insurance = decodeUnicodeEscape(insurance)
		log.Printf("[extractFromContextData] id=%s Insurance: %s", propertyID, property.Insurance)
	}

	// Extract room layout image URL
	if roomLayoutImageURL, ok := contextData["RoomLayoutImageUrl"].(string); ok && roomLayoutImageURL != "" {
		property.RoomLayoutImageURL = roomLayoutImageURL
		log.Printf("[extractFromContextData] id=%s RoomLayoutImageURL: %s", propertyID, property.RoomLayoutImageURL)
	}

	// Final summary
	rentVal := "NULL"
	if property.Rent != nil {
		rentVal = fmt.Sprintf("%d", *property.Rent)
	}
	ageVal := "NULL"
	if property.BuildingAge != nil {
		ageVal = fmt.Sprintf("%d", *property.BuildingAge)
	}
	log.Printf("[extractFromContextData] id=%s FINAL: title=%q rent=%s station=%q age=%s address=%q",
		propertyID, property.Title, rentVal, property.Station, ageVal, property.Address)
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

// normalizeFloorPlan normalizes Japanese floor plan names to standard codes
func normalizeFloorPlan(floorPlan string) string {
	if floorPlan == "" {
		return ""
	}

	// First check if it's already in standard format (1R, 1K, 1DK, etc.)
	if matched, _ := regexp.MatchString(`^[0-9]?[SLDK]+$`, floorPlan); matched {
		return floorPlan
	}

	// Normalize Japanese variations to standard codes
	floorPlan = strings.TrimSpace(floorPlan)

	// Extract base floor plan type
	if strings.Contains(floorPlan, "ワンルーム") || strings.Contains(floorPlan, "1R") {
		return "1R"
	}
	if strings.Contains(floorPlan, "1K") {
		return "1K"
	}
	if strings.Contains(floorPlan, "1DK") {
		return "1DK"
	}
	if strings.Contains(floorPlan, "1LDK") {
		return "1LDK"
	}
	if strings.Contains(floorPlan, "1SDK") {
		return "1SDK"
	}
	if strings.Contains(floorPlan, "1SLDK") {
		return "1SLDK"
	}
	if strings.Contains(floorPlan, "2K") {
		return "2K"
	}
	if strings.Contains(floorPlan, "2DK") {
		return "2DK"
	}
	if strings.Contains(floorPlan, "2LDK") {
		return "2LDK"
	}
	if strings.Contains(floorPlan, "2SDK") {
		return "2SDK"
	}
	if strings.Contains(floorPlan, "2SLDK") {
		return "2SLDK"
	}
	if strings.Contains(floorPlan, "3K") {
		return "3K"
	}
	if strings.Contains(floorPlan, "3DK") {
		return "3DK"
	}
	if strings.Contains(floorPlan, "3LDK") {
		return "3LDK"
	}
	if strings.Contains(floorPlan, "3SDK") {
		return "3SDK"
	}
	if strings.Contains(floorPlan, "3SLDK") {
		return "3SLDK"
	}
	if strings.Contains(floorPlan, "4K") {
		return "4K"
	}
	if strings.Contains(floorPlan, "4DK") {
		return "4DK"
	}
	if strings.Contains(floorPlan, "4LDK") {
		return "4LDK"
	}

	// If no match, return as-is (but log it)
	log.Printf("[normalizeFloorPlan] Unknown floor plan format: %s", floorPlan)
	return floorPlan
}

// cleanTitle removes unwanted text from property titles
// Removes "Yahoo不動産" and common suffixes like " - Yahoo!不動産"
func cleanTitle(title string) string {
	title = strings.TrimSpace(title)

	// Remove "Yahoo不動産" and variations
	patterns := []string{
		"Yahoo不動産",
		"Yahoo!不動産",
		"yahoo不動産",
		"YAHOO不動産",
	}

	for _, pattern := range patterns {
		title = strings.ReplaceAll(title, pattern, "")
	}

	// Remove common separators and trailing text after them
	// Examples: "物件名 - Yahoo不動産" -> "物件名"
	separators := []string{" - ", " | ", "｜", " 【"}
	for _, sep := range separators {
		if idx := strings.Index(title, sep); idx > 0 {
			title = title[:idx]
		}
	}

	// Remove leading/trailing whitespace and special characters
	title = strings.TrimSpace(title)
	title = strings.Trim(title, "- |｜【】")
	title = strings.TrimSpace(title)

	return title
}

// normalizeBuildingType normalizes Japanese building types to English codes
// Uses structure (構造) as primary classifier since RC = mansion, wooden = apartment
func normalizeBuildingType(buildingType, structure string) string {
	buildingType = strings.TrimSpace(buildingType)
	structure = strings.TrimSpace(structure)

	// Primary classification: Use structure/construction material
	// 鉄筋コンクリート (RC) = mansion, 木造 (wooden) = apartment
	if structure != "" {
		if strings.Contains(structure, "鉄筋コンクリート") || strings.Contains(structure, "RC") ||
		   strings.Contains(structure, "鉄骨鉄筋コンクリート") || strings.Contains(structure, "SRC") {
			return "mansion"
		}
		if strings.Contains(structure, "木造") {
			return "apartment"
		}
		// 軽量鉄骨 (light steel) is typically used for apartments
		if strings.Contains(structure, "軽量鉄骨") {
			return "apartment"
		}
		// 鉄骨造 (steel frame) without RC is typically apartment/light construction
		if strings.Contains(structure, "鉄骨造") && !strings.Contains(structure, "鉄筋") {
			return "apartment"
		}
	}

	// Fallback: Use building type label if structure doesn't give clear answer
	if buildingType != "" {
		if strings.Contains(buildingType, "マンション") {
			return "mansion"
		}
		if strings.Contains(buildingType, "アパート") {
			return "apartment"
		}
		if strings.Contains(buildingType, "一戸建") || strings.Contains(buildingType, "戸建") {
			return "house"
		}
		if strings.Contains(buildingType, "テラスハウス") {
			return "terrace_house"
		}
		if strings.Contains(buildingType, "タウンハウス") {
			return "town_house"
		}
		if strings.Contains(buildingType, "シェアハウス") {
			return "share_house"
		}
	}

	// If already in English, return lowercase
	if buildingType != "" {
		return strings.ToLower(buildingType)
	}
	return ""
}

// normalizeFacilities converts Yahoo facility codes to English keys for filtering
func normalizeFacilities(facilitiesJSON string) string {
	if facilitiesJSON == "" {
		return ""
	}

	// Parse the JSON array of Yahoo codes
	var yahooCodes []string
	if err := json.Unmarshal([]byte(facilitiesJSON), &yahooCodes); err != nil {
		log.Printf("[normalizeFacilities] Failed to parse facilities JSON: %v", err)
		return facilitiesJSON // Return as-is if can't parse
	}

	// Map Yahoo codes to English keys
	// Based on common Yahoo Real Estate facility codes
	codeMap := map[string]string{
		"011": "bath_toilet_separate",     // バス・トイレ別
		"012": "bath_toilet_separate",     // バストイレ別 (alternate)
		"013": "independent_washbasin",    // 独立洗面台
		"014": "independent_washbasin",    // 独立洗面台 (alternate)
		"001": "auto_lock",                // オートロック
		"003": "second_floor_plus",        // 2階以上
		"005": "south_facing",             // 南向き
		"017": "reheating_bath",           // 追い焚き風呂
		"030": "walk_in_closet",           // ウォークインクローゼット
		"022": "flooring",                 // フローリング
		"002": "pet_friendly",             // ペット可
		"pet": "pet_friendly",             // ペット可 (alternate)
	}

	normalizedKeys := make(map[string]bool)
	for _, code := range yahooCodes {
		if key, ok := codeMap[code]; ok {
			normalizedKeys[key] = true
		}
	}

	// Convert back to JSON array
	var keys []string
	for key := range normalizedKeys {
		keys = append(keys, key)
	}

	if len(keys) == 0 {
		return "" // No recognized facilities
	}

	result, err := json.Marshal(keys)
	if err != nil {
		log.Printf("[normalizeFacilities] Failed to marshal normalized keys: %v", err)
		return facilitiesJSON
	}

	return string(result)
}

// normalizeFacilitiesFromLabels converts Japanese facility labels to English keys
// This handles cases where facilities are presented as text labels instead of codes
func normalizeFacilitiesFromLabels(labels []string) []string {
	if len(labels) == 0 {
		return []string{}
	}

	// Map Japanese labels to English keys
	// Comprehensive mapping based on Yahoo Real Estate facility labels
	labelMap := map[string]string{
		// Bath/Toilet
		"バス・トイレ独立": "bath_toilet_separate",
		"バストイレ別":    "bath_toilet_separate",
		"バス・トイレ別":  "bath_toilet_separate",
		"独立洗面台":      "independent_washbasin",
		"洗面台":         "washbasin",
		"追い焚き風呂":    "reheating_bath",
		"追い焚き":       "reheating_bath",
		"シャワー":       "shower",
		"トイレ":        "toilet",
		"風呂":          "bath",
		"浴室乾燥機":      "bathroom_dryer",
		"給湯":          "hot_water",

		// Security
		"オートロック":          "auto_lock",
		"防犯カメラ":           "security_camera",
		"TVモニター付きインターホン": "tv_intercom",
		"ディンプルキー":         "dimple_key",
		"日中管理":            "daytime_manager",

		// Floor/Position
		"2階以上":   "second_floor_plus",
		"最上階":    "top_floor",
		"角部屋":    "corner_room",
		"南向き":    "south_facing",
		"ベランダ":   "balcony",
		"バルコニー":  "balcony",

		// Kitchen
		"コンロ2口以上":      "two_burner_stove",
		"システムキッチン":     "system_kitchen",
		"カウンターキッチン":    "counter_kitchen",
		"IHコンロ":         "ih_stove",
		"ガスコンロ":        "gas_stove",

		// Interior
		"フローリング":        "flooring",
		"室内洗濯機置き場":      "indoor_laundry_space",
		"洗濯機置き場":        "laundry_space",
		"エアコン":          "air_conditioner",
		"床暖房":           "floor_heating",
		"ウォークインクローゼット": "walk_in_closet",
		"シューズボックス":      "shoe_box",
		"収納":            "storage",
		"クローゼット":        "closet",

		// Utilities
		"都市ガス":      "city_gas",
		"光ファイバー":    "fiber_internet",
		"光回線":       "fiber_internet",
		"インターネット":   "internet",
		"インターネット対応": "internet_ready",
		"BS":         "bs_antenna",
		"CS":         "cs_antenna",

		// Pet
		"ペット可":   "pet_friendly",
		"ペット相談":  "pet_negotiable",

		// Payment
		"カード決済可": "card_payment",

		// Building Type - Condominium type (分譲タイプ) is a facility feature, not building type
		"分譲タイプ": "condominium_type",

		// Other
		"エレベーター":    "elevator",
		"駐輪場":       "bicycle_parking",
		"バイク置き場":    "motorcycle_parking",
		"駐車場":       "car_parking",
		"宅配ボックス":    "delivery_box",
		"ゴミ出し24時間":   "24h_trash",
		"ゴミ置き場":     "garbage_area",
		"タイル張り":     "tile_exterior",
		"タイル":       "tile_exterior",
	}

	normalizedKeys := make(map[string]bool)
	for _, label := range labels {
		label = strings.TrimSpace(label)
		if key, ok := labelMap[label]; ok {
			normalizedKeys[key] = true
		} else {
			// Try partial matching for compound labels
			for japLabel, engKey := range labelMap {
				if strings.Contains(label, japLabel) {
					normalizedKeys[engKey] = true
					break
				}
			}
		}
	}

	// Convert to sorted array for consistency
	keys := make([]string, 0, len(normalizedKeys))
	for key := range normalizedKeys {
		keys = append(keys, key)
	}

	return keys
}

// extractPopularFeatureLabels extracts facility labels from "人気の特徴・設備" section
func extractPopularFeatureLabels(doc *goquery.Document) []string {
	var labels []string
	seen := make(map[string]bool)

	// Find the "人気の特徴・設備" section by text content
	doc.Find("*").Each(func(_ int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if text != "人気の特徴・設備" {
			return
		}

		// Try to find the scope: prefer next sibling block (h3 の次の div), fallback to parent
		scope := s.Next()
		if scope.Length() == 0 {
			// Fallback: try parent's next sibling
			scope = s.Parent().Next()
		}
		if scope.Length() == 0 {
			// Last resort: use parent
			scope = s.Parent()
		}

		// Extract from img alt attributes (most reliable)
		// Only extract items that are NOT disabled (skip --disabled class)
		scope.Find("img[alt]").Each(func(_ int, img *goquery.Selection) {
			// Check if parent has --disabled class
			parent := img.Parent()
			if parent.Length() > 0 {
				if class, exists := parent.Attr("class"); exists && strings.Contains(class, "--disabled") {
					return // Skip disabled items
				}
			}

			if alt, exists := img.Attr("alt"); exists {
				alt = strings.TrimSpace(html.UnescapeString(alt))
				if alt != "" && !seen[alt] {
					labels = append(labels, alt)
					seen[alt] = true
				}
			}
		})

		// Also try to extract from nearby text (backup)
		scope.Find("li, span, div").Each(func(_ int, elem *goquery.Selection) {
			txt := strings.TrimSpace(elem.Text())
			// Skip if too long (likely not a feature label)
			if len(txt) > 0 && len(txt) < 30 && !seen[txt] {
				// Only add if it looks like a facility label
				if strings.Contains(txt, "階") || strings.Contains(txt, "付") ||
				   strings.Contains(txt, "可") || strings.Contains(txt, "別") ||
				   strings.Contains(txt, "台") || strings.Contains(txt, "ロック") {
					labels = append(labels, txt)
					seen[txt] = true
				}
			}
		})
	})

	return labels
}

// extractCategoryFacilityLabels extracts facilities from category lines like "バス・トイレ シャワー / トイレ / 風呂"
func extractCategoryFacilityLabels(doc *goquery.Document) []string {
	facilityCategories := []string{
		"バス・トイレ", "キッチン", "室内設備", "収納", "通信",
		"ベランダ", "セキュリティ", "入居条件", "位置", "その他",
	}

	var labels []string
	seen := make(map[string]bool)

	// Get all text content and split by lines
	text := doc.Text()
	text = strings.ReplaceAll(text, "\r\n", "\n")
	lines := strings.Split(text, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check if line starts with a facility category
		for _, cat := range facilityCategories {
			if strings.HasPrefix(line, cat) {
				// Extract the part after category name
				rest := strings.TrimSpace(strings.TrimPrefix(line, cat))

				// Split by "/" to get individual facilities
				parts := strings.Split(rest, "/")
				for _, part := range parts {
					label := strings.TrimSpace(part)
					if label != "" && !seen[label] {
						labels = append(labels, label)
						seen[label] = true
					}
				}
				break
			}
		}
	}

	return labels
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

// StationAccess represents a single station access point
type StationAccess struct {
	StationName string
	LineName    string
	WalkMinutes int
	SortOrder   int
}

// extractStations extracts all station access points from the document
// Returns array of StationAccess with sort_order 1, 2, 3...
func extractStations(doc *goquery.Document) []StationAccess {
	var stations []StationAccess
	sortOrder := 1

	// Find all station access entries
	doc.Find("li.DetailSummaryTable__access").Each(func(_ int, s *goquery.Selection) {
		// Get the full text content
		text := s.Text()
		text = strings.TrimSpace(text)

		// Normalize spaces (both full-width and half-width)
		text = strings.ReplaceAll(text, "　", " ")
		text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")

		// Extract station name from <a class="_SummaryStation">
		stationName := ""
		s.Find("a._SummaryStation").Each(func(_ int, a *goquery.Selection) {
			stationName = strings.TrimSpace(a.Text())
		})

		if stationName == "" {
			return // Skip if no station name found
		}

		// Parse the text after station name: "/東京メトロ丸ノ内線 徒歩6分"
		// Pattern: stationName + "/" + lineName + " 徒歩" + minutes + "分"

		// Find the position after station name
		afterStation := strings.Replace(text, stationName, "", 1)
		afterStation = strings.TrimSpace(afterStation)

		// Remove leading "/" if present
		afterStation = strings.TrimPrefix(afterStation, "/")
		afterStation = strings.TrimSpace(afterStation)

		lineName := ""
		walkMinutes := 0

		// Try to extract walk time: "徒歩(\d+)分"
		walkRe := regexp.MustCompile(`徒歩\s*([0-9]+)\s*分`)
		walkMatches := walkRe.FindStringSubmatch(afterStation)
		if len(walkMatches) > 1 {
			if val, err := strconv.Atoi(walkMatches[1]); err == nil {
				// Validate: walk time should be reasonable (1-60 minutes)
				if val >= 1 && val <= 120 {
					walkMinutes = val
				}
			}
		}

		// Extract line name: everything before "徒歩" or "バス" etc.
		// Split by common transportation keywords
		lineRe := regexp.MustCompile(`^(.+?)\s*(?:徒歩|バス|車)`)
		lineMatches := lineRe.FindStringSubmatch(afterStation)
		if len(lineMatches) > 1 {
			lineName = strings.TrimSpace(lineMatches[1])
		} else {
			// If no transportation keyword found, use the whole text (edge case)
			lineName = afterStation
		}

		// Clean up line name: remove trailing "/" or spaces
		lineName = strings.Trim(lineName, "/ 　")

		// If walkMinutes is 0, it means this is non-walk access (bus, etc.)
		// Still save it with walk_minutes = 0 and preserve line_name/station_name

		stations = append(stations, StationAccess{
			StationName: stationName,
			LineName:    lineName,
			WalkMinutes: walkMinutes,
			SortOrder:   sortOrder,
		})

		sortOrder++
	})

	return stations
}

// convertStationsToModels converts StationAccess to PropertyStation models
func convertStationsToModels(propertyID string, stations []StationAccess) []models.PropertyStation {
	result := make([]models.PropertyStation, 0, len(stations))
	for _, s := range stations {
		ps := models.PropertyStation{
			PropertyID:  propertyID,
			StationName: s.StationName,
			LineName:    s.LineName,
			WalkMinutes: s.WalkMinutes,
			SortOrder:   s.SortOrder,
		}
		result = append(result, ps)
	}
	return result
}

// applyStationCompatibility copies the primary station (sort_order=1) to legacy fields
// for backward compatibility with existing code
func applyStationCompatibility(prop *models.Property, stations []StationAccess) {
	if len(stations) == 0 {
		return
	}

	// Use the first station (sort_order=1)
	s0 := stations[0]
	if s0.StationName != "" {
		prop.Station = s0.StationName
	}

	// Only update walk_time if it's a valid walk time (> 0)
	if s0.WalkMinutes > 0 {
		prop.WalkTime = &s0.WalkMinutes
	}
}

// GetLastStations returns the stations from the last scrape operation
func (s *Scraper) GetLastStations() []StationAccess {
	return s.lastStations
}

// GetLastStationsAsModels returns the stations from the last scrape as PropertyStation models
func (s *Scraper) GetLastStationsAsModels(propertyID string) []models.PropertyStation {
	return convertStationsToModels(propertyID, s.lastStations)
}

// GetLastImages returns the image URLs from the last scrape
func (s *Scraper) GetLastImages() []string {
	return s.lastImages
}

// GetLastImagesAsModels returns the images from the last scrape as PropertyImage models
func (s *Scraper) GetLastImagesAsModels(propertyID string) []models.PropertyImage {
	images := make([]models.PropertyImage, 0, len(s.lastImages))
	for i, imageURL := range s.lastImages {
		images = append(images, models.PropertyImage{
			PropertyID: propertyID,
			ImageURL:   imageURL,
			SortOrder:  i,
		})
	}
	return images
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

	// For Yahoo Real Estate detail pages, KEEP the trailing slash
	// (removing it causes 301 redirects which can fail scraping)
	// For other URLs (list pages, search pages), remove trailing slash
	isDetailPage := strings.Contains(parsedURL.Path, "/rent/detail/")
	if !isDetailPage && len(parsedURL.Path) > 1 && strings.HasSuffix(parsedURL.Path, "/") {
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

// extractAllImageURLsFromJSON extracts all image URLs from embedded JSON data and HTML
// Returns an array of image URLs from realestate-pctr domain (deduplicated and limited to ~20 images)
func extractAllImageURLsFromJSON(html string) []string {
	var imageURLs []string

	// Extract all realestate-pctr image URLs from HTML (both JSON and img tags)
	// This pattern matches the full URL format used by Yahoo Real Estate
	imgPattern := regexp.MustCompile(`https://realestate-pctr\.c\.yimg\.jp/[A-Za-z0-9_\-]+`)
	matches := imgPattern.FindAllString(html, -1)

	// Use URL signature (chars 150-350) to identify unique images
	// Same image in different sizes/crops shares this middle section of the URL
	seenSignatures := make(map[string]bool)
	seenURLs := make(map[string]bool)

	for _, url := range matches {
		// Unescape if needed
		cleanURL := strings.ReplaceAll(url, `\/`, `/`)

		// Skip if we've seen this exact URL
		if seenURLs[cleanURL] {
			continue
		}
		seenURLs[cleanURL] = true

		// Extract signature from middle of URL (chars 140-240)
		// This part identifies the actual image, while the end varies for different sizes
		var signature string
		if len(cleanURL) > 240 {
			signature = cleanURL[140:240]
		} else if len(cleanURL) > 140 {
			signature = cleanURL[140:]
		} else {
			signature = cleanURL
		}

		// Only add if we haven't seen this image signature before
		if !seenSignatures[signature] {
			imageURLs = append(imageURLs, cleanURL)
			seenSignatures[signature] = true

			// Limit to ~20 unique images (typical for Yahoo Real Estate properties)
			if len(imageURLs) >= 20 {
				break
			}
		}
	}

	log.Printf("[extractAllImageURLsFromJSON] Found %d unique images (from %d total URLs, %d completely unique)", len(imageURLs), len(matches), len(seenURLs))

	// If we found images, return them
	if len(imageURLs) > 0 {
		return imageURLs
	}

	// Fallback: Try extracting from ResizedExternalImageUrls JSON field
	resizedExternalRe := regexp.MustCompile(`"ResizedExternalImageUrls"\s*:\s*\[([^\]]+)\]`)
	if matches := resizedExternalRe.FindStringSubmatch(html); len(matches) > 1 {
		log.Printf("[extractAllImageURLsFromJSON] Fallback: Found ResizedExternalImageUrls, extracting URLs...")
		urlRe := regexp.MustCompile(`"Url"\s*:\s*"([^"]+)"`)
		urlMatches := urlRe.FindAllStringSubmatch(matches[1], -1)
		for _, urlMatch := range urlMatches {
			if len(urlMatch) > 1 {
				imageURL := strings.ReplaceAll(urlMatch[1], `\/`, `/`)
				if !seenURLs[imageURL] {
					imageURLs = append(imageURLs, imageURL)
					seenURLs[imageURL] = true
				}
			}
		}
	}

	// Final fallback: ExternalImageUrl (single image)
	if len(imageURLs) == 0 {
		if externalImageURL := extractExternalImageFromJSON(html); externalImageURL != "" {
			imageURLs = append(imageURLs, externalImageURL)
			log.Printf("[extractAllImageURLsFromJSON] Fallback: Found 1 image from ExternalImageUrl")
		}
	}

	if len(imageURLs) == 0 {
		log.Printf("[extractAllImageURLsFromJSON] No images found")
	}

	return imageURLs
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

	// Validate length (Yahoo property IDs can be 40-48 characters with optional prefix)
	// Accept 44-48 characters (including _0000 prefix formats)
	if len(propertyID) < 40 || len(propertyID) > 48 {
		return "", fmt.Errorf("unexpected property ID length %d (expected 40-48): %s", len(propertyID), propertyID)
	}

	return propertyID, nil
}
