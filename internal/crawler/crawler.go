package crawler

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yourusername/go-web-crawler/internal/config"
	"github.com/yourusername/go-web-crawler/internal/parser"
	"github.com/yourusername/go-web-crawler/internal/storage"
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
type Crawler struct {
	config       *config.Config
	queue        *Queue
	visited      *storage.VisitedStore
	parser       *parser.HTMLParser
	robotsCache  *RobotsCache
	rateLimiter  *rate.Limiter
	logger       *slog.Logger

	// Statistics
	stats        Stats
	totalDuration atomic.Int64

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

	var robotsCache *RobotsCache
	if cfg.RespectRobotsTxt {
		robotsCache = NewRobotsCache(cfg.UserAgent, cfg.RequestTimeout)
	}

	// Create rate limiter (requests per second)
	limiter := rate.NewLimiter(rate.Limit(cfg.RateLimit), cfg.RateLimit)

	c := &Crawler{
		config:      cfg,
		queue:       NewQueue(),
		visited:     storage.NewVisitedStore(),
		parser:      htmlParser,
		robotsCache: robotsCache,
		rateLimiter: limiter,
		logger:      logger,
		ctx:         ctx,
		cancel:      cancel,
		results:     make([]WorkerResult, 0),
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

	w := NewWorker(id, c.config.RequestTimeout, c.config.UserAgent,
		c.config.MaxRetries, c.config.RetryDelay, c.logger)

	lastDomainAccess := make(map[string]time.Time)
	var lastAccessMu sync.Mutex

	for {
		// Check if we should stop
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		// Check if we've reached max pages
		if c.config.MaxPages > 0 && atomic.LoadInt64(&c.stats.PagesVisited) >= int64(c.config.MaxPages) {
			return
		}

		// Try to get a URL from the queue
		item, ok := c.queue.TryDequeue()
		if !ok {
			// Queue is empty, check if we should exit
			time.Sleep(100 * time.Millisecond)

			// If queue is still empty and all workers are idle, we're done
			if c.queue.IsEmpty() {
				return
			}
			continue
		}

		// Check depth limit
		if c.config.MaxDepth > 0 && item.Depth > c.config.MaxDepth {
			continue
		}

		// Check robots.txt if enabled
		if c.config.RespectRobotsTxt && c.robotsCache != nil {
			if !c.robotsCache.IsAllowed(c.ctx, item.URL) {
				c.logger.Debug("URL disallowed by robots.txt",
					"worker", id,
					"url", item.URL,
				)
				continue
			}
		}

		// Apply rate limiting
		if err := c.rateLimiter.Wait(c.ctx); err != nil {
			return // Context cancelled
		}

		// Apply politeness delay for same domain
		if c.config.PolitenessDelay > 0 {
			domain, err := parser.GetDomain(item.URL)
			if err == nil {
				lastAccessMu.Lock()
				if lastAccess, exists := lastDomainAccess[domain]; exists {
					elapsed := time.Since(lastAccess)
					if elapsed < c.config.PolitenessDelay {
						time.Sleep(c.config.PolitenessDelay - elapsed)
					}
				}
				lastDomainAccess[domain] = time.Now()
				lastAccessMu.Unlock()
			}
		}

		// Process the URL
		result := w.Process(c.ctx, item.URL, item.Depth)

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