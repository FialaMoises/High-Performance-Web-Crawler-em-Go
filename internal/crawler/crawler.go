package crawler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/FialaMoises/go-web-crawler/internal/config"
	"github.com/FialaMoises/go-web-crawler/internal/parser"
	"github.com/FialaMoises/go-web-crawler/internal/storage"
	"golang.org/x/time/rate"
)

// Stats holds crawler statistics
type Stats struct {
	PagesVisited   int64
	PagesFailed    int64
	LinksFound     int64
	StartTime      time.Time
	EndTime        time.Time
	AverageDuration time.Duration
}

// Crawler is the main crawler engine
// IMPORTANT: 64-bit atomic fields must be at the top of the struct for proper alignment on 32-bit systems
type Crawler struct {
	// Statistics (64-bit fields MUST come first for atomic operations)
	stats        Stats
	totalDuration atomic.Int64

	// Configuration and dependencies
	config       *config.Config
	queue        URLQueue
	visited      URLStore
	parser       URLParser
	robotsCache  RobotsChecker
	rateLimiter  *rate.Limiter
	logger       *slog.Logger
	httpClient   *http.Client
	authenticator *Authenticator
	jsRenderer    *JSRenderer

	// Control
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup

	// Results
	results      []WorkerResult
	resultsMu    sync.Mutex
}

// NewCrawler creates a new Crawler instance
func NewCrawler(cfg *config.Config, logger *slog.Logger) (*Crawler, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	htmlParser, err := parser.NewHTMLParser(cfg.StartURL)
	if err != nil {
		return nil, fmt.Errorf("create parser: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Create HTTP client with cookie jar (for session management)
	jar, err := cookiejar.New(nil)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("create cookie jar: %w", err)
	}

	httpClient := &http.Client{
		Jar:     jar,
		Timeout: cfg.RequestTimeout,
	}

	var robotsCache *RobotsCache
	if cfg.RespectRobotsTxt {
		robotsCache = NewRobotsCache(cfg.UserAgent, cfg.RequestTimeout)
	}

	// Create rate limiter (requests per second)
	limiter := rate.NewLimiter(rate.Limit(cfg.RateLimit), cfg.RateLimit)

	// Create authenticator if required
	var auth *Authenticator
	if cfg.RequiresAuth {
		auth = NewAuthenticator(cfg, httpClient, logger)
	}

	// Create JavaScript renderer if required
	var jsRenderer *JSRenderer
	if cfg.RenderJS {
		jsRenderer = NewJSRenderer(logger, cfg.JSTimeout)
	}

	c := &Crawler{
		config:        cfg,
		queue:         NewQueue(),
		visited:       storage.NewVisitedStore(),
		parser:        htmlParser,
		robotsCache:   robotsCache,
		rateLimiter:   limiter,
		logger:        logger,
		httpClient:    httpClient,
		authenticator: auth,
		jsRenderer:    jsRenderer,
		ctx:           ctx,
		cancel:        cancel,
		results:       make([]WorkerResult, 0),
	}

	return c, nil
}

// Start begins the crawling process
func (c *Crawler) Start() error {
	c.stats.StartTime = time.Now()
	c.logger.Info("Starting crawler",
		"start_url", c.config.StartURL,
		"max_depth", c.config.MaxDepth,
		"max_pages", c.config.MaxPages,
		"num_workers", c.config.NumWorkers,
	)

	// Perform authentication if required
	if c.config.RequiresAuth && c.authenticator != nil {
		c.logger.Info("Authentication required, logging in...")
		if err := c.authenticator.Login(c.ctx); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
		c.logger.Info("Authentication successful")
	}

	// Add seed URL to queue
	c.queue.Enqueue(URLItem{URL: c.config.StartURL, Depth: 0})
	c.visited.Add(c.config.StartURL)

	// Start worker pool
	for i := 0; i < c.config.NumWorkers; i++ {
		c.wg.Add(1)
		go c.worker(i)
	}

	// Wait for all workers to finish
	c.wg.Wait()

	c.stats.EndTime = time.Now()

	// Calculate average duration
	if c.stats.PagesVisited > 0 {
		avgNanos := c.totalDuration.Load() / c.stats.PagesVisited
		c.stats.AverageDuration = time.Duration(avgNanos)
	}

	c.logger.Info("Crawling completed",
		"pages_visited", c.stats.PagesVisited,
		"pages_failed", c.stats.PagesFailed,
		"links_found", c.stats.LinksFound,
		"duration", c.stats.EndTime.Sub(c.stats.StartTime),
		"avg_page_duration", c.stats.AverageDuration,
	)

	return nil
}

// Stop gracefully stops the crawler
func (c *Crawler) Stop() {
	c.logger.Info("Stopping crawler...")
	c.cancel()
}

// worker is the worker goroutine that processes URLs from the queue
func (c *Crawler) worker(id int) {
	defer c.wg.Done()

	w := NewWorker(id, c.httpClient, c.config.UserAgent,
		c.config.MaxRetries, c.config.RetryDelay, c.logger, c.authenticator,
		c.jsRenderer, c.config.RenderJS)

	lastDomainAccess := make(map[string]time.Time)
	var lastAccessMu sync.Mutex

	for {
		if !c.shouldContinue() {
			return
		}

		item, ok := c.getNextItem()
		if !ok {
			continue
		}

		if !c.shouldProcessItem(item, id) {
			continue
		}

		c.applyPoliteness(item.URL, &lastDomainAccess, &lastAccessMu)

		result := w.Process(c.ctx, item.URL, item.Depth)
		c.handleResult(result, item)
	}
}

// shouldContinue checks if the worker should continue processing
func (c *Crawler) shouldContinue() bool {
	select {
	case <-c.ctx.Done():
		return false
	default:
	}

	if c.config.MaxPages > 0 && atomic.LoadInt64(&c.stats.PagesVisited) >= int64(c.config.MaxPages) {
		return false
	}

	return true
}

// getNextItem attempts to get the next URL from the queue
func (c *Crawler) getNextItem() (URLItem, bool) {
	item, ok := c.queue.TryDequeue()
	if !ok {
		time.Sleep(100 * time.Millisecond)
		if c.queue.IsEmpty() {
			return URLItem{}, false
		}
		return URLItem{}, false
	}
	return item, true
}

// shouldProcessItem checks if an item should be processed
func (c *Crawler) shouldProcessItem(item URLItem, workerID int) bool {
	// Check depth limit
	if c.config.MaxDepth > 0 && item.Depth > c.config.MaxDepth {
		return false
	}

	// Check robots.txt if enabled
	if c.config.RespectRobotsTxt && c.robotsCache != nil {
		if !c.robotsCache.IsAllowed(c.ctx, item.URL) {
			c.logger.Debug("URL disallowed by robots.txt",
				"worker", workerID,
				"url", item.URL,
			)
			return false
		}
	}

	// Apply rate limiting
	if err := c.rateLimiter.Wait(c.ctx); err != nil {
		return false
	}

	return true
}

// applyPoliteness applies politeness delay for the same domain
func (c *Crawler) applyPoliteness(targetURL string, lastAccess *map[string]time.Time, mu *sync.Mutex) {
	if c.config.PolitenessDelay <= 0 {
		return
	}

	domain, err := parser.GetDomain(targetURL)
	if err != nil {
		return
	}

	mu.Lock()
	defer mu.Unlock()

	if lastTime, exists := (*lastAccess)[domain]; exists {
		elapsed := time.Since(lastTime)
		if elapsed < c.config.PolitenessDelay {
			time.Sleep(c.config.PolitenessDelay - elapsed)
		}
	}
	(*lastAccess)[domain] = time.Now()
}

// handleResult processes a worker result
func (c *Crawler) handleResult(result WorkerResult, item URLItem) {
	// Update statistics
	if result.Success {
		atomic.AddInt64(&c.stats.PagesVisited, 1)
		atomic.AddInt64(&c.stats.LinksFound, int64(len(result.Links)))
		c.totalDuration.Add(result.Duration.Nanoseconds())
	} else {
		atomic.AddInt64(&c.stats.PagesFailed, 1)
	}

	// Store result
	c.resultsMu.Lock()
	c.results = append(c.results, result)
	c.resultsMu.Unlock()

	// Process discovered links
	if result.Success {
		c.processLinks(result.Links, item.URL, item.Depth+1)
	}
}

// processLinks processes discovered links and adds them to the queue
func (c *Crawler) processLinks(links []string, baseURL string, depth int) {
	baseURLParsed, err := url.Parse(baseURL)
	if err != nil {
		return
	}

	var newItems []URLItem

	for _, link := range links {
		// Parse and resolve relative URLs
		linkURL, err := url.Parse(link)
		if err != nil {
			continue
		}

		absoluteURL := baseURLParsed.ResolveReference(linkURL)
		normalizedURL := c.normalizeURL(absoluteURL)

		if normalizedURL == "" {
			continue
		}

		// Check same domain restriction
		if c.config.SameDomainOnly && !c.parser.IsSameDomain(normalizedURL) {
			continue
		}

		// Check if already visited (deduplicate)
		if c.visited.Add(normalizedURL) {
			newItems = append(newItems, URLItem{
				URL:   normalizedURL,
				Depth: depth,
			})
		}
	}

	if len(newItems) > 0 {
		c.queue.EnqueueBatch(newItems)
	}
}

// normalizeURL normalizes a URL
func (c *Crawler) normalizeURL(u *url.URL) string {
	// Remove fragment
	u.Fragment = ""

	// Only accept http/https
	if u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}

	return u.String()
}

// GetResults returns all crawl results
func (c *Crawler) GetResults() []WorkerResult {
	c.resultsMu.Lock()
	defer c.resultsMu.Unlock()

	// Return a copy
	results := make([]WorkerResult, len(c.results))
	copy(results, c.results)
	return results
}

// GetStats returns crawler statistics
func (c *Crawler) GetStats() Stats {
	return c.stats
}

// GetVisitedURLs returns all visited URLs
func (c *Crawler) GetVisitedURLs() []string {
	return c.visited.GetAll()
}