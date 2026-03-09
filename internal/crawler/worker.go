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
	id            int
	client        *http.Client
	userAgent     string
	maxRetries    int
	retryDelay    time.Duration
	logger        *slog.Logger
	authenticator *Authenticator
	jsRenderer    *JSRenderer
	renderJS      bool
}

// NewWorker creates a new Worker
func NewWorker(id int, client *http.Client, userAgent string, maxRetries int, retryDelay time.Duration, logger *slog.Logger, auth *Authenticator, renderer *JSRenderer, renderJS bool) *Worker {
	return &Worker{
		id:            id,
		client:        client, // Use shared client (with cookies)
		userAgent:     userAgent,
		maxRetries:    maxRetries,
		retryDelay:    retryDelay,
		logger:        logger,
		authenticator: auth,
		jsRenderer:    renderer,
		renderJS:      renderJS,
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
	// If JavaScript rendering is enabled, use headless browser
	if w.renderJS && w.jsRenderer != nil {
		return w.fetchWithJSRender(ctx, url)
	}

	// Otherwise, use standard HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", w.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	// Add authentication headers if authenticator is present
	if w.authenticator != nil {
		w.authenticator.AddAuthToRequest(req)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	// Check for authentication errors and attempt re-login
	if w.authenticator != nil && (resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden) {
		w.logger.Warn("Authentication error detected, attempting re-login",
			"worker", w.id,
			"url", url,
			"status", resp.StatusCode,
		)

		// Attempt re-authentication
		wasAuthError, authErr := w.authenticator.HandleAuthError(ctx, resp.StatusCode)
		if wasAuthError && authErr == nil {
			// Re-authentication successful, retry the request
			w.logger.Info("Re-authentication successful, retrying request", "worker", w.id)

			// Create new request
			req, err = http.NewRequestWithContext(ctx, "GET", url, nil)
			if err != nil {
				return nil, fmt.Errorf("create retry request: %w", err)
			}

			req.Header.Set("User-Agent", w.userAgent)
			req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
			req.Header.Set("Accept-Language", "en-US,en;q=0.9")
			w.authenticator.AddAuthToRequest(req)

			// Retry request
			resp, err = w.client.Do(req)
			if err != nil {
				return nil, fmt.Errorf("http retry request: %w", err)
			}
			defer resp.Body.Close()
		} else if wasAuthError && authErr != nil {
			return nil, fmt.Errorf("re-authentication failed: %w", authErr)
		}
	}

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

// fetchWithJSRender fetches a page using headless browser with JavaScript rendering
func (w *Worker) fetchWithJSRender(ctx context.Context, url string) (*goquery.Document, error) {
	// Get authentication token if available
	var authToken string
	if w.authenticator != nil {
		authToken = w.authenticator.GetToken()
	}

	w.logger.Debug("Rendering page with JavaScript",
		"worker", w.id,
		"url", url,
		"has_auth", authToken != "",
	)

	// Render the page with JavaScript
	html, err := w.jsRenderer.RenderPage(ctx, url, authToken)
	if err != nil {
		return nil, fmt.Errorf("render page with JS: %w", err)
	}

	// Parse the rendered HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("parse rendered html: %w", err)
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