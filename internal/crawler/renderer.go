package crawler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/chromedp/chromedp"
)

// JSRenderer handles JavaScript rendering using headless Chrome
type JSRenderer struct {
	logger  *slog.Logger
	timeout time.Duration
}

// NewJSRenderer creates a new JavaScript renderer
func NewJSRenderer(logger *slog.Logger, timeout time.Duration) *JSRenderer {
	return &JSRenderer{
		logger:  logger,
		timeout: timeout,
	}
}

// RenderPage renders a page with JavaScript and returns the final HTML
func (r *JSRenderer) RenderPage(ctx context.Context, url string, authToken string) (string, error) {
	// Create a timeout context
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	// Create Chrome context
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-web-security", true),
		chromedp.Flag("disable-features", "IsolateOrigins,site-per-process"),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx, opts...)
	defer allocCancel()

	// Create browser context
	browserCtx, browserCancel := chromedp.NewContext(allocCtx)
	defer browserCancel()

	var html string
	var err error

	// Navigate to the page and get rendered HTML
	tasks := chromedp.Tasks{
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Set authorization header if token exists
			if authToken != "" {
				return chromedp.Run(ctx,
					chromedp.ActionFunc(func(ctx context.Context) error {
						// Add authorization header to all requests
						expr := fmt.Sprintf(`
							(function() {
								const originalFetch = window.fetch;
								window.fetch = function(...args) {
									if (args[1] === undefined) {
										args[1] = {};
									}
									if (args[1].headers === undefined) {
										args[1].headers = {};
									}
									args[1].headers['Authorization'] = 'Bearer %s';
									return originalFetch.apply(this, args);
								};
							})();
						`, authToken)
						return chromedp.Evaluate(expr, nil).Do(ctx)
					}),
				)
			}
			return nil
		}),
		chromedp.Navigate(url),
		// Wait for network to be idle
		chromedp.Sleep(2 * time.Second),
		// Get the rendered HTML
		chromedp.OuterHTML("html", &html),
	}

	err = chromedp.Run(browserCtx, tasks)
	if err != nil {
		return "", fmt.Errorf("failed to render page: %w", err)
	}

	r.logger.Debug("Page rendered with JavaScript",
		"url", url,
		"html_length", len(html),
	)

	return html, nil
}