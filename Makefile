# Makefile for Go Web Crawler

.PHONY: help build run docker-build docker-run clean test

help: ## Show this help message
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build the Go binary
	go build -o crawler ./cmd/crawler

run: ## Run the crawler locally
	go run ./cmd/crawler -url https://books.toscrape.com -depth 2 -pages 50

docker-build: ## Build Docker image
	docker build -t go-web-crawler:latest .

docker-run: ## Run crawler in Docker
	docker run -v $$(pwd)/output:/app/output go-web-crawler:latest -url https://books.toscrape.com -depth 3 -pages 100

docker-compose-up: ## Run with docker-compose
	docker-compose --profile examples up crawler-books

clean: ## Clean build artifacts and output
	rm -f crawler
	rm -rf output/*

test: ## Run tests (when available)
	go test -v ./...

deps: ## Download dependencies
	go mod download

tidy: ## Tidy go.mod
	go mod tidy

fmt: ## Format code
	go fmt ./...

vet: ## Run go vet
	go vet ./...

lint: fmt vet ## Run linters

all: clean deps build ## Build everything from scratch