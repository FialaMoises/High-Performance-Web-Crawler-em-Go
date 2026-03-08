#!/bin/bash

# Quick run script for Go Web Crawler
# Usage: ./run.sh [URL] [optional: depth] [optional: pages]

set -e

# Default values
URL="${1:-https://books.toscrape.com}"
DEPTH="${2:-3}"
PAGES="${3:-100}"
WORKERS="${4:-10}"

echo "=============================================="
echo "  Go Web Crawler - Quick Run"
echo "=============================================="
echo "URL:     $URL"
echo "Depth:   $DEPTH"
echo "Pages:   $PAGES"
echo "Workers: $WORKERS"
echo "=============================================="
echo ""

# Create output directory if it doesn't exist
mkdir -p output

# Check if Docker is available
if command -v docker &> /dev/null; then
    echo "🐳 Running with Docker..."

    # Build image if it doesn't exist
    if [[ "$(docker images -q go-web-crawler:latest 2> /dev/null)" == "" ]]; then
        echo "📦 Building Docker image..."
        docker build -t go-web-crawler:latest .
    fi

    # Run crawler
    docker run -v "$(pwd)/output:/app/output" go-web-crawler:latest \
        -url "$URL" \
        -depth "$DEPTH" \
        -pages "$PAGES" \
        -workers "$WORKERS" \
        -format both \
        -log-level info

elif command -v go &> /dev/null; then
    echo "🔧 Running with Go..."

    # Download dependencies if needed
    if [ ! -d "vendor" ]; then
        echo "📥 Downloading dependencies..."
        go mod download
    fi

    # Run crawler
    go run ./cmd/crawler \
        -url "$URL" \
        -depth "$DEPTH" \
        -pages "$PAGES" \
        -workers "$WORKERS" \
        -format both \
        -log-level info
else
    echo "❌ Error: Neither Docker nor Go found!"
    echo "Please install Docker or Go to run the crawler."
    exit 1
fi

echo ""
echo "✅ Done! Check the output directory for results."