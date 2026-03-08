package config

import "time"

// Config holds all crawler configuration
type Config struct {
	// StartURL is the initial URL to begin crawling
	StartURL string

	// MaxDepth is the maximum depth to crawl (0 = unlimited)
	MaxDepth int

	// MaxPages is the maximum number of pages to crawl (0 = unlimited)
	MaxPages int

	// NumWorkers is the number of concurrent workers
	NumWorkers int

	// RequestTimeout is the timeout for HTTP requests
	RequestTimeout time.Duration

	// RateLimit is the maximum requests per second
	RateLimit int

	// SameDomainOnly restricts crawling to the same domain as StartURL
	SameDomainOnly bool

	// RespectRobotsTxt enables robots.txt compliance
	RespectRobotsTxt bool

	// UserAgent is the User-Agent header for HTTP requests
	UserAgent string

	// MaxRetries is the maximum number of retries for failed requests
	MaxRetries int

	// RetryDelay is the initial delay between retries (exponential backoff)
	RetryDelay time.Duration

	// PolitenessDelay is the minimum delay between requests to the same domain
	PolitenessDelay time.Duration

	// OutputFormat specifies export format: "json", "csv", or "both"
	OutputFormat string

	// OutputPath is the directory to save results
	OutputPath string
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig(startURL string) *Config {
	return &Config{
		StartURL:         startURL,
		MaxDepth:         3,
		MaxPages:         1000,
		NumWorkers:       10,
		RequestTimeout:   10 * time.Second,
		RateLimit:        100,
		SameDomainOnly:   true,
		RespectRobotsTxt: true,
		UserAgent:        "GoWebCrawler/1.0 (+https://github.com/FialaMoises/go-web-crawler)",
		MaxRetries:       3,
		RetryDelay:       1 * time.Second,
		PolitenessDelay:  500 * time.Millisecond,
		OutputFormat:     "both",
		OutputPath:       "./output",
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.StartURL == "" {
		return &ConfigError{Field: "StartURL", Message: "cannot be empty"}
	}
	if c.NumWorkers < 1 {
		return &ConfigError{Field: "NumWorkers", Message: "must be at least 1"}
	}
	if c.RequestTimeout < 1*time.Second {
		return &ConfigError{Field: "RequestTimeout", Message: "must be at least 1 second"}
	}
	return nil
}

// ConfigError represents a configuration validation error
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return "config error: " + e.Field + " " + e.Message
}