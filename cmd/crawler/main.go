package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/FialaMoises/go-web-crawler/internal/config"
	"github.com/FialaMoises/go-web-crawler/internal/crawler"
	"github.com/FialaMoises/go-web-crawler/internal/export"
)

var (
	version = "1.0.0"
	banner  = `
╔══════════════════════════════════════════════╗
║   High Performance Web Crawler in Go        ║
║   Version: %s                         ║
╚══════════════════════════════════════════════╝
`
)

func main() {
	// Parse command line flags
	var (
		startURL      = flag.String("url", "", "Starting URL to crawl (required)")
		maxDepth      = flag.Int("depth", 3, "Maximum crawl depth")
		maxPages      = flag.Int("pages", 1000, "Maximum pages to crawl")
		numWorkers    = flag.Int("workers", 10, "Number of concurrent workers")
		rateLimit     = flag.Int("rate", 100, "Maximum requests per second")
		sameDomain    = flag.Bool("same-domain", true, "Only crawl URLs from the same domain")
		respectRobots = flag.Bool("robots", true, "Respect robots.txt")
		outputFormat  = flag.String("format", "both", "Output format: json, csv, or both")
		outputPath    = flag.String("output", "./output", "Output directory for results")
		timeout       = flag.Int("timeout", 10, "HTTP request timeout in seconds")
		maxRetries    = flag.Int("retries", 3, "Maximum number of retries for failed requests")
		logLevel      = flag.String("log-level", "info", "Log level: debug, info, warn, error")
		showVersion   = flag.Bool("version", false, "Show version and exit")
	)

	flag.Parse()

	// Show version
	if *showVersion {
		fmt.Printf("go-web-crawler version %s\n", version)
		return
	}

	// Print banner
	fmt.Printf(banner, version)

	// Validate required flags
	if *startURL == "" {
		fmt.Println("Error: -url flag is required")
		flag.Usage()
		os.Exit(1)
	}

	// Setup logger
	logger := setupLogger(*logLevel)

	// Create configuration
	cfg := &config.Config{
		StartURL:         *startURL,
		MaxDepth:         *maxDepth,
		MaxPages:         *maxPages,
		NumWorkers:       *numWorkers,
		RequestTimeout:   time.Duration(*timeout) * time.Second,
		RateLimit:        *rateLimit,
		SameDomainOnly:   *sameDomain,
		RespectRobotsTxt: *respectRobots,
		UserAgent:        fmt.Sprintf("GoWebCrawler/%s (+https://github.com/FialaMoises/go-web-crawler)", version),
		MaxRetries:       *maxRetries,
		RetryDelay:       1 * time.Second,
		PolitenessDelay:  500 * time.Millisecond,
		OutputFormat:     *outputFormat,
		OutputPath:       *outputPath,
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		logger.Error("Invalid configuration", "error", err)
		os.Exit(1)
	}

	// Print configuration
	printConfig(cfg)

	// Create crawler
	c, err := crawler.NewCrawler(cfg, logger)
	if err != nil {
		logger.Error("Failed to create crawler", "error", err)
		os.Exit(1)
	}

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\n\nReceived interrupt signal. Shutting down gracefully...")
		c.Stop()
	}()

	// Start crawling
	fmt.Println("\n🚀 Starting crawl...")

	startTime := time.Now()
	if err := c.Start(); err != nil {
		logger.Error("Crawler failed", "error", err)
		os.Exit(1)
	}

	// Print summary
	stats := c.GetStats()
	printSummary(stats, time.Since(startTime))

	// Export results
	fmt.Println("\n📊 Exporting results...")

	exporter := export.NewExporter(cfg.OutputPath)

	results := c.GetResults()
	if err := exporter.Export(results, stats, cfg.StartURL, cfg.OutputFormat); err != nil {
		logger.Error("Failed to export results", "error", err)
	}

	// Export URL list
	urls := c.GetVisitedURLs()
	if err := exporter.ExportURLList(urls); err != nil {
		logger.Error("Failed to export URL list", "error", err)
	}

	fmt.Println("\n✨ Crawl completed successfully!")
}

// setupLogger creates and configures a structured logger
func setupLogger(level string) *slog.Logger {
	var logLevel slog.Level

	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	handler := slog.NewTextHandler(os.Stdout, opts)
	return slog.New(handler)
}

// printConfig prints the crawler configuration
func printConfig(cfg *config.Config) {
	fmt.Println("📋 Configuration:")
	fmt.Printf("  Start URL:        %s\n", cfg.StartURL)
	fmt.Printf("  Max Depth:        %d\n", cfg.MaxDepth)
	fmt.Printf("  Max Pages:        %d\n", cfg.MaxPages)
	fmt.Printf("  Workers:          %d\n", cfg.NumWorkers)
	fmt.Printf("  Rate Limit:       %d req/s\n", cfg.RateLimit)
	fmt.Printf("  Same Domain Only: %t\n", cfg.SameDomainOnly)
	fmt.Printf("  Respect robots.txt: %t\n", cfg.RespectRobotsTxt)
	fmt.Printf("  Output Format:    %s\n", cfg.OutputFormat)
	fmt.Printf("  Output Path:      %s\n", cfg.OutputPath)
}

// printSummary prints crawl statistics
func printSummary(stats crawler.Stats, duration time.Duration) {
	separator := strings.Repeat("=", 60)
	fmt.Println("\n" + separator)
	fmt.Println("📈 Crawl Summary")
	fmt.Println(separator)
	fmt.Printf("  Total Duration:      %s\n", duration.Round(time.Millisecond))
	fmt.Printf("  Pages Visited:       %d\n", stats.PagesVisited)
	fmt.Printf("  Pages Failed:        %d\n", stats.PagesFailed)
	fmt.Printf("  Total Links Found:   %d\n", stats.LinksFound)
	fmt.Printf("  Avg Page Duration:   %s\n", stats.AverageDuration.Round(time.Millisecond))

	if stats.PagesVisited > 0 {
		successRate := float64(stats.PagesVisited) / float64(stats.PagesVisited+stats.PagesFailed) * 100
		fmt.Printf("  Success Rate:        %.2f%%\n", successRate)

		pagesPerSecond := float64(stats.PagesVisited) / duration.Seconds()
		fmt.Printf("  Pages/Second:        %.2f\n", pagesPerSecond)
	}

	fmt.Println(separator)
}