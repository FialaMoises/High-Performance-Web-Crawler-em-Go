package crawler

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// WorkerResult represents the result of processing a URL
type WorkerResult struct {
	URL       string
	Links     []string
	Depth     int
	Success   bool
	Error     error
	Duration  time.Duration
	Timestamp time.Time
}

// Worker processes URLs from the queue
type Worker struct {
	id         int
	client     *http.Client
	userAgent  string
	maxRetries int
	retryDelay time.Duration
	logger     *slog.Logger
}

// NewWorker creates a new Worker
func NewWorker(id int, timeout time.Duration, userAgent string, maxRetries int, retryDelay time.Duration, logger *slog.Logger) *Worker {
	return &Worker{
		id: id,
		client: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Allow up to 10 redirects
				if len(via) >= 10 {
					return fmt.Errorf("stopped after 10 redirects")
				}
				return nil
			},
		},
		userAgent:  userAgent,
		maxRetries: maxRetries,
		retryDelay: retryDelay,
		logger:     logger,
	}
}

// Process fetches a URL and extracts links
func (w *Worker) Process(ctx context.Context, url string, depth int) WorkerResult {
	start := time.Now()
	result := WorkerResult{
		URL:       url,
		Depth:     depth,
		Timestamp: start,
	}

	// Try with retries and exponential backoff
	var doc *goquery.Document
	var err error

	for attempt := 0; attempt <= w.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := time.Duration(attempt) * w.retryDelay
			w.logger.Debug("Retrying request",
				"worker", w.id,
				"url", url,
				"attempt", attempt,
				"backoff", backoff,
			)

			select {
			case <-ctx.Done():
				result.Error = ctx.Err()
				result.Duration = time.Since(start)
				return result
			case <-time.After(backoff):
			}
		}

		doc, err = w.fetchDocument(ctx, url)
		if err == nil {
			break
		}

		// Don't retry on context cancellation
		if ctx.Err() != nil {
			result.Error = ctx.Err()
			result.Duration = time.Since(start)
			return result
		}
	}

	if err != nil {
		w.logger.Warn("Failed to fetch URL after retries",
			"worker", w.id,
			"url", url,
			"error", err,
			"attempts", w.maxRetries+1,
		)
		result.Error = err
		result.Duration = time.Since(start)
		return result
	}

	// Extract links from the document
	links := w.extractLinks(doc)

	result.Links = links
	result.Success = true
	result.Duration = time.Since(start)

	w.logger.Info("Successfully processed URL",
		"worker", w.id,
		"url", url,
		"links_found", len(links),
		"duration", result.Duration,
	)

	return result
}

// fetchDocument fetches and parses an HTML document
func (w *Worker) fetchDocument(ctx context.Context, url string) (*goquery.Document, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", w.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := w.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http status: %d", resp.StatusCode)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(strings.ToLower(contentType), "text/html") {
		return nil, fmt.Errorf("not html content: %s", contentType)
	}

	// Limit response size to 10MB
	limitedReader := io.LimitReader(resp.Body, 10*1024*1024)

	// Parse HTML
	doc, err := goquery.NewDocumentFromReader(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	return doc, nil
}

// extractLinks extracts all links from a document
func (w *Worker) extractLinks(doc *goquery.Document) []string {
	var links []string
	seen := make(map[string]bool)

	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists || href == "" {
			return
		}

		// Skip anchors and javascript links
		if strings.HasPrefix(href, "#") || strings.HasPrefix(href, "javascript:") {
			return
		}

		// Deduplicate
		if !seen[href] {
			seen[href] = true
			links = append(links, href)
		}
	})

	return links
}