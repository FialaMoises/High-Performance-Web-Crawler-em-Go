package crawler

import (
	"context"

	"github.com/PuerkitoBio/goquery"
)

// URLParser defines the interface for parsing links from HTML documents
type URLParser interface {
	// ParseLinks extracts all valid links from an HTML document
	ParseLinks(doc *goquery.Document, currentURL string) ([]string, error)

	// IsSameDomain checks if a URL belongs to the same domain as the base URL
	IsSameDomain(targetURL string) bool
}

// URLStore defines the interface for tracking visited URLs
type URLStore interface {
	// Add marks a URL as visited
	// Returns true if the URL was newly added, false if it was already visited
	Add(url string) bool

	// Has checks if a URL has been visited
	Has(url string) bool

	// Count returns the number of visited URLs
	Count() int

	// GetAll returns a copy of all visited URLs
	GetAll() []string

	// Clear removes all visited URLs
	Clear()
}

// RobotsChecker defines the interface for checking robots.txt compliance
type RobotsChecker interface {
	// IsAllowed checks if the given URL is allowed to be crawled
	IsAllowed(ctx context.Context, urlStr string) bool

	// Clear removes all cached robots.txt data
	Clear()
}

// URLQueue defines the interface for managing the URL queue
type URLQueue interface {
	// Enqueue adds a URL item to the queue
	Enqueue(item URLItem)

	// EnqueueBatch adds multiple URL items to the queue
	EnqueueBatch(items []URLItem)

	// TryDequeue attempts to dequeue without blocking
	// Returns false if queue is empty
	TryDequeue() (URLItem, bool)

	// Len returns the current length of the queue
	Len() int

	// IsEmpty checks if the queue is empty
	IsEmpty() bool

	// Clear removes all items from the queue
	Clear()
}
