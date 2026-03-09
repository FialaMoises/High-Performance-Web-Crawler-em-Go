# Multi-stage Dockerfile for Go Web Crawler
# Stage 1: Build
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -o /crawler \
    ./cmd/crawler

# Stage 2: Runtime
FROM alpine:latest

# Install runtime dependencies including Chromium for JavaScript rendering
RUN apk --no-cache add \
    ca-certificates \
    tzdata \
    chromium \
    chromium-chromedriver \
    nss \
    freetype \
    harfbuzz \
    ttf-freefont \
    && rm -rf /var/cache/apk/*

# Set Chromium environment variables
ENV CHROME_BIN=/usr/bin/chromium-browser \
    CHROME_PATH=/usr/lib/chromium/

# Create non-root user
RUN addgroup -g 1000 crawler && \
    adduser -D -u 1000 -G crawler crawler

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /crawler /app/crawler

# Create output directory
RUN mkdir -p /app/output && \
    chown -R crawler:crawler /app

# Switch to non-root user
USER crawler

# Set default command
ENTRYPOINT ["/app/crawler"]

# Default help message if no args provided
CMD ["--help"]