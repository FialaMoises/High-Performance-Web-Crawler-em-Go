package crawler

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/temoto/robotstxt"
)

// RobotsCache caches robots.txt data for different domains
type RobotsCache struct {
	mu      sync.RWMutex
	cache   map[string]*robotstxt.RobotsData
	client  *http.Client
	userAgent string
}

// NewRobotsCache creates a new RobotsCache
func NewRobotsCache(userAgent string, timeout time.Duration) *RobotsCache {
	return &RobotsCache{
		cache:     make(map[string]*robotstxt.RobotsData),
		client:    &http.Client{Timeout: timeout},
		userAgent: userAgent,
	}
}

// IsAllowed checks if the given URL is allowed to be crawled
func (rc *RobotsCache) IsAllowed(ctx context.Context, urlStr string) bool {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false
	}

	domain := parsedURL.Scheme + "://" + parsedURL.Host

	// Check cache first
	rc.mu.RLock()
	robotsData, exists := rc.cache[domain]
	rc.mu.RUnlock()

	if !exists {
		// Fetch robots.txt
		robotsData = rc.fetchRobotsTxt(ctx, domain)

		// Cache the result
		rc.mu.Lock()
		rc.cache[domain] = robotsData
		rc.mu.Unlock()
	}

	// If we couldn't fetch robots.txt, allow by default
	if robotsData == nil {
		return true
	}

	// Check if URL is allowed for our user agent
	return robotsData.TestAgent(parsedURL.Path, rc.userAgent)
}

// fetchRobotsTxt fetches and parses robots.txt for a domain
func (rc *RobotsCache) fetchRobotsTxt(ctx context.Context, domain string) *robotstxt.RobotsData {
	robotsURL := domain + "/robots.txt"

	req, err := http.NewRequestWithContext(ctx, "GET", robotsURL, nil)
	if err != nil {
		return nil
	}

	req.Header.Set("User-Agent", rc.userAgent)

	resp, err := rc.client.Do(req)
	if err != nil {
		// Network error, allow by default
		return nil
	}
	defer resp.Body.Close()

	// If robots.txt doesn't exist (404), allow everything
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}

	// If we got a different error code, be conservative and allow by default
	if resp.StatusCode != http.StatusOK {
		return nil
	}

	// Parse robots.txt
	data, err := robotstxt.FromResponse(resp)
	if err != nil {
		// Parse error, allow by default
		return nil
	}

	return data
}

// GetCrawlDelay returns the crawl delay specified in robots.txt for the user agent
func (rc *RobotsCache) GetCrawlDelay(ctx context.Context, urlStr string) time.Duration {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return 0
	}

	domain := parsedURL.Scheme + "://" + parsedURL.Host

	rc.mu.RLock()
	robotsData, exists := rc.cache[domain]
	rc.mu.RUnlock()

	if !exists {
		robotsData = rc.fetchRobotsTxt(ctx, domain)
		rc.mu.Lock()
		rc.cache[domain] = robotsData
		rc.mu.Unlock()
	}

	if robotsData == nil {
		return 0
	}

	group := robotsData.FindGroup(rc.userAgent)
	if group == nil {
		return 0
	}

	return group.CrawlDelay
}

// Clear removes all cached robots.txt data
func (rc *RobotsCache) Clear() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.cache = make(map[string]*robotstxt.RobotsData)
}

// Stats returns statistics about the robots cache
func (rc *RobotsCache) Stats() string {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	return fmt.Sprintf("Cached robots.txt for %d domains", len(rc.cache))
}