package storage

import "sync"

// VisitedStore keeps track of visited URLs in a thread-safe manner
type VisitedStore struct {
	mu      sync.RWMutex
	visited map[string]bool
}

// NewVisitedStore creates a new VisitedStore
func NewVisitedStore() *VisitedStore {
	return &VisitedStore{
		visited: make(map[string]bool),
	}
}

// Add marks a URL as visited
// Returns true if the URL was newly added, false if it was already visited
func (vs *VisitedStore) Add(url string) bool {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	if vs.visited[url] {
		return false
	}

	vs.visited[url] = true
	return true
}

// Has checks if a URL has been visited
func (vs *VisitedStore) Has(url string) bool {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	return vs.visited[url]
}

// Count returns the number of visited URLs
func (vs *VisitedStore) Count() int {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	return len(vs.visited)
}

// GetAll returns a copy of all visited URLs
func (vs *VisitedStore) GetAll() []string {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	urls := make([]string, 0, len(vs.visited))
	for url := range vs.visited {
		urls = append(urls, url)
	}

	return urls
}

// Clear removes all visited URLs
func (vs *VisitedStore) Clear() {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	vs.visited = make(map[string]bool)
}