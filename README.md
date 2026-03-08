# 🚀 High Performance Web Crawler in Go

A concurrent web crawler built in Go capable of crawling thousands of pages efficiently using goroutines and worker pools.

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat&logo=docker)](https://www.docker.com/)


## ✨ Features

- **🔥 High Performance**: Uses goroutines and worker pools for concurrent crawling
- **🤖 robots.txt Compliance**: Respects robots.txt rules automatically
- **🎯 Smart Rate Limiting**: Configurable requests per second with politeness delay
- **🔄 Retry Logic**: Exponential backoff for failed requests
- **🌐 Domain Filtering**: Option to crawl only same-domain URLs
- **📊 Multiple Export Formats**: JSON, CSV, and plain text
- **🛡️ Thread-Safe**: Concurrent-safe URL deduplication
- **📈 Real-time Statistics**: Track pages visited, links found, and performance metrics
- **🐳 Docker Ready**: Fully containerized with multi-stage builds
- **📝 Structured Logging**: Clean, structured logs with configurable levels

## 🏗️ Architecture

```
go-web-crawler/
│
├── cmd/
│   └── crawler/
│       └── main.go              # CLI entry point
│
├── internal/
│   ├── config/
│   │   └── config.go            # Configuration management
│   │
│   ├── crawler/
│   │   ├── crawler.go           # Main crawler engine
│   │   ├── worker.go            # Worker pool implementation
│   │   ├── queue.go             # Thread-safe URL queue
│   │   └── robots.go            # robots.txt parser
│   │
│   ├── parser/
│   │   └── html_parser.go       # HTML parsing & link extraction
│   │
│   ├── storage/
│   │   └── visited.go           # Thread-safe visited URLs store
│   │
│   └── export/
│       └── exporter.go          # Result export (JSON/CSV)
│
├── Dockerfile                   # Multi-stage Docker build
├── docker-compose.yml           # Docker Compose configuration
└── README.md
```

## 🚀 Quick Start

### Using Docker (Recommended)

1. **Build the Docker image:**
```bash
docker build -t go-web-crawler .
```

2. **Run the crawler:**
```bash
docker run -v $(pwd)/output:/app/output go-web-crawler \
  -url https://books.toscrape.com \
  -depth 3 \
  -pages 100 \
  -workers 10
```

### Using Docker Compose

1. **Run example crawls:**
```bash
# Crawl books.toscrape.com
docker-compose --profile examples up crawler-books

# Crawl quotes.toscrape.com
docker-compose --profile examples up crawler-quotes

# Run with debug logging
docker-compose --profile debug up crawler-debug
```

2. **Custom crawl:**
Edit `docker-compose.yml` and modify the command parameters, then:
```bash
docker-compose up crawler
```

### Local Build (without Docker)

1. **Prerequisites:**
   - Go 1.21 or higher

2. **Install dependencies:**
```bash
go mod download
```

3. **Build:**
```bash
go build -o crawler ./cmd/crawler
```

4. **Run:**
```bash
./crawler -url https://books.toscrape.com -depth 3 -pages 100
```

## 📖 Usage

### Command Line Options

```
Usage of crawler:
  -url string
        Starting URL to crawl (required)
  -depth int
        Maximum crawl depth (default 3)
  -pages int
        Maximum pages to crawl (default 1000)
  -workers int
        Number of concurrent workers (default 10)
  -rate int
        Maximum requests per second (default 100)
  -same-domain
        Only crawl URLs from the same domain (default true)
  -robots
        Respect robots.txt (default true)
  -format string
        Output format: json, csv, or both (default "both")
  -output string
        Output directory for results (default "./output")
  -timeout int
        HTTP request timeout in seconds (default 10)
  -retries int
        Maximum number of retries for failed requests (default 3)
  -log-level string
        Log level: debug, info, warn, error (default "info")
  -version
        Show version and exit
```

### Examples

**Basic crawl:**
```bash
docker run -v $(pwd)/output:/app/output go-web-crawler \
  -url https://books.toscrape.com
```

**Fast crawl with many workers:**
```bash
docker run -v $(pwd)/output:/app/output go-web-crawler \
  -url https://books.toscrape.com \
  -workers 20 \
  -rate 200 \
  -pages 500
```

**Deep crawl with debug logging:**
```bash
docker run -v $(pwd)/output:/app/output go-web-crawler \
  -url https://example.com \
  -depth 5 \
  -log-level debug
```

**Export only JSON:**
```bash
docker run -v $(pwd)/output:/app/output go-web-crawler \
  -url https://quotes.toscrape.com \
  -format json
```

## 📊 Output

The crawler generates three types of output files in the specified output directory:

1. **JSON Results** (`crawl_results_TIMESTAMP.json`):
```json
{
  "summary": {
    "start_url": "https://books.toscrape.com",
    "pages_visited": 250,
    "pages_failed": 5,
    "links_found": 1523,
    "total_duration": "45.2s",
    "average_duration": "180ms"
  },
  "results": [
    {
      "url": "https://books.toscrape.com/index.html",
      "depth": 0,
      "success": true,
      "links_found": 52,
      "duration": "245ms",
      "timestamp": "2024-01-15T10:30:00Z"
    }
  ]
}
```

2. **CSV Results** (`crawl_results_TIMESTAMP.csv`):
```csv
URL,Depth,Success,Links Found,Duration,Timestamp,Error
https://books.toscrape.com/index.html,0,true,52,245ms,2024-01-15T10:30:00Z,
```

3. **Visited URLs List** (`visited_urls_TIMESTAMP.txt`):
```
https://books.toscrape.com/index.html
https://books.toscrape.com/catalogue/page-2.html
https://books.toscrape.com/catalogue/book_1/index.html
...
```

## 🧰 Tech Stack

- **Go 1.21+**: Modern Go with generics and improved performance
- **goquery**: jQuery-like HTML parsing
- **robotstxt**: robots.txt parsing and compliance
- **golang.org/x/time/rate**: Rate limiting
- **slog**: Structured logging (Go 1.21+)

## 🎯 Recommended Test Sites

Safe sites designed for web scraping practice:

| Site | URL | Best For |
|------|-----|----------|
| Books to Scrape | https://books.toscrape.com | E-commerce, pagination, depth testing |
| Quotes to Scrape | https://quotes.toscrape.com | JavaScript rendering, authentication |
| Scrape This Site | https://scrapethissite.com | Various scraping scenarios |

## 🔧 Configuration

### Worker Pool Optimization

The number of workers affects crawling speed:

- **Low (5-10)**: Conservative, respectful crawling
- **Medium (10-20)**: Balanced performance
- **High (20-50)**: Maximum speed (use with caution)

### Rate Limiting

Adjust based on target site capacity:

- **Conservative**: 10-50 req/s
- **Moderate**: 50-100 req/s
- **Aggressive**: 100-200 req/s (may get blocked)

## 📈 Performance

Typical performance on a modern laptop:

- **Pages/second**: 10-50 (depending on target site)
- **Memory usage**: ~50-100 MB
- **CPU usage**: 10-30% (scales with workers)

## 🐳 Docker Details

### Multi-Stage Build

The Dockerfile uses a multi-stage build for optimal image size:

- **Builder stage**: ~800 MB (includes Go compiler)
- **Runtime stage**: ~15 MB (minimal Alpine image)

### Non-Root User

The container runs as a non-root user (`crawler:crawler`) for security.

## 🤝 Contributing

Contributions are welcome! Feel free to:

- Report bugs
- Suggest features
- Submit pull requests

## 📝 License


## 🙏 Acknowledgments

- Built with [goquery](https://github.com/PuerkitoBio/goquery)
- robots.txt parsing by [robotstxt](https://github.com/temoto/robotstxt)
- Inspired by the Go concurrency patterns

## 📧 Contact

**Moises Fiala**
- GitHub: [@FialaMoises](https://github.com/FialaMoises)
- Project: [go-web-crawler](https://github.com/FialaMoises/go-web-crawler)

---

**⭐ If you find this project useful, please consider giving it a star!**

---

## 🎓 Learning Resources

This project demonstrates:

- ✅ Goroutines and channels
- ✅ Worker pool pattern
- ✅ Context for cancellation
- ✅ Rate limiting
- ✅ HTTP client best practices
- ✅ HTML parsing
- ✅ Concurrent data structures
- ✅ Structured logging
- ✅ Docker multi-stage builds

Perfect for portfolio and technical interviews!
