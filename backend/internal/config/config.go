package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Database      DatabaseConfig      `yaml:"database"`
	Search        SearchConfig        `yaml:"search"`
	Scraper       ScraperConfig       `yaml:"scraper"`
	RateLimit     RateLimitConfig     `yaml:"rate_limit"`
	ErrorHandling ErrorHandlingConfig `yaml:"error_handling"`
	UserAgent     string              `yaml:"user_agent"`
	Logging       LoggingConfig       `yaml:"logging"`
	Timezone      string              `yaml:"timezone"`
}

// DatabaseConfig contains database settings
type DatabaseConfig struct {
	Type     string         `yaml:"type"`
	MySQL    MySQLConfig    `yaml:"mysql"`
	Postgres PostgresConfig `yaml:"postgres"`
}

// MySQLConfig contains MySQL connection settings
type MySQLConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

// PostgresConfig contains PostgreSQL connection settings
type PostgresConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
	SSLMode  string `yaml:"sslmode"`
}

// SearchConfig contains search engine settings
type SearchConfig struct {
	Meilisearch MeilisearchConfig `yaml:"meilisearch"`
}

// MeilisearchConfig contains Meilisearch connection settings
type MeilisearchConfig struct {
	Host   string `yaml:"host"`
	APIKey string `yaml:"api_key"`
}

// ScraperConfig contains scraper-specific settings
type ScraperConfig struct {
	RequestDelaySeconds int    `yaml:"request_delay_seconds"`
	TimeoutSeconds      int    `yaml:"timeout_seconds"`
	MaxRetries          int    `yaml:"max_retries"`
	RetryDelaySeconds   int    `yaml:"retry_delay_seconds"`
	MaxRequestsPerDay   int    `yaml:"max_requests_per_day"`
	StopOnError         bool   `yaml:"stop_on_error"`
	ConcurrentLimit     int    `yaml:"concurrent_limit"`
	DailyRunEnabled     bool   `yaml:"daily_run_enabled"`
	DailyRunTime        string `yaml:"daily_run_time"`
	ListPageLimit       int    `yaml:"list_page_limit"`
}

// RateLimitConfig contains rate limiting settings
type RateLimitConfig struct {
	Enabled            bool `yaml:"enabled"`
	RequestsPerMinute  int  `yaml:"requests_per_minute"`
	RequestsPerHour    int  `yaml:"requests_per_hour"`
}

// ErrorHandlingConfig contains error handling settings
type ErrorHandlingConfig struct {
	RetryOnNetworkError bool `yaml:"retry_on_network_error"`
	RetryOn5xx          bool `yaml:"retry_on_5xx"`
	RetryOn4xx          bool `yaml:"retry_on_4xx"`
	LogErrors           bool `yaml:"log_errors"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level        string `yaml:"level"`
	LogRequests  bool   `yaml:"log_requests"`
	LogResponses bool   `yaml:"log_responses"`
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		Scraper: ScraperConfig{
			RequestDelaySeconds: 2,
			TimeoutSeconds:      30,
			MaxRetries:          3,
			RetryDelaySeconds:   2,
			MaxRequestsPerDay:   5000,
			StopOnError:         true,
			ConcurrentLimit:     1,
			DailyRunEnabled:     false,
			DailyRunTime:        "02:00",
			ListPageLimit:       50,
		},
		RateLimit: RateLimitConfig{
			Enabled:           true,
			RequestsPerMinute: 30,
			RequestsPerHour:   1800,
		},
		ErrorHandling: ErrorHandlingConfig{
			RetryOnNetworkError: true,
			RetryOn5xx:          true,
			RetryOn4xx:          false,
			LogErrors:           true,
		},
		UserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
		Logging: LoggingConfig{
			Level:        "info",
			LogRequests:  true,
			LogResponses: false,
		},
	}
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(filepath string) (*Config, error) {
	// Start with default config
	config := DefaultConfig()

	// If file doesn't exist, return default config
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return config, nil
	}

	// Read file
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}

// GetRequestDelay returns the request delay as a duration
func (c *ScraperConfig) GetRequestDelay() time.Duration {
	return time.Duration(c.RequestDelaySeconds) * time.Second
}

// GetTimeout returns the timeout as a duration
func (c *ScraperConfig) GetTimeout() time.Duration {
	return time.Duration(c.TimeoutSeconds) * time.Second
}

// GetRetryDelay returns the retry delay as a duration
func (c *ScraperConfig) GetRetryDelay() time.Duration {
	return time.Duration(c.RetryDelaySeconds) * time.Second
}

// ToScraperConfig converts config.ScraperConfig to scraper.ScraperConfig
// Note: This returns a map of configuration values that can be used by the scraper package
func (c *ScraperConfig) ToScraperParams() map[string]interface{} {
	return map[string]interface{}{
		"timeout":       c.GetTimeout(),
		"max_retries":   c.MaxRetries,
		"retry_delay":   c.GetRetryDelay(),
		"request_delay": c.GetRequestDelay(),
	}
}
