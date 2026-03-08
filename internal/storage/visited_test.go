package storage

import (
	"sync"
	"testing"
)

func TestVisitedStore_Add(t *testing.T) {
	vs := NewVisitedStore()

	// Test adding new URL
	if !vs.Add("https://example.com") {
		t.Error("Add should return true for new URL")
	}

	// Test adding duplicate URL
	if vs.Add("https://example.com") {
		t.Error("Add should return false for duplicate URL")
	}
}

func TestVisitedStore_Has(t *testing.T) {
	vs := NewVisitedStore()

	// Test URL not visited
	if vs.Has("https://example.com") {
		t.Error("Has should return false for unvisited URL")
	}

	// Test URL visited
	vs.Add("https://example.com")
	if !vs.Has("https://example.com") {
		t.Error("Has should return true for visited URL")
	}
}

func TestVisitedStore_Count(t *testing.T) {
	vs := NewVisitedStore()

	if vs.Count() != 0 {
		t.Errorf("Initial count should be 0, got %d", vs.Count())
	}

	vs.Add("https://example.com")
	vs.Add("https://example.org")
	vs.Add("https://example.com") // duplicate

	if vs.Count() != 2 {
		t.Errorf("Count should be 2, got %d", vs.Count())
	}
}

func TestVisitedStore_GetAll(t *testing.T) {
	vs := NewVisitedStore()

	vs.Add("https://example.com")
	vs.Add("https://example.org")

	urls := vs.GetAll()
	if len(urls) != 2 {
		t.Errorf("GetAll should return 2 URLs, got %d", len(urls))
	}

	// Check both URLs are present
	found := make(map[string]bool)
	for _, url := range urls {
		found[url] = true
	}

	if !found["https://example.com"] || !found["https://example.org"] {
		t.Error("GetAll should return all visited URLs")
	}
}

func TestVisitedStore_Clear(t *testing.T) {
	vs := NewVisitedStore()

	vs.Add("https://example.com")
	vs.Add("https://example.org")

	vs.Clear()

	if vs.Count() != 0 {
		t.Errorf("Count after Clear should be 0, got %d", vs.Count())
	}

	if vs.Has("https://example.com") {
		t.Error("Has should return false after Clear")
	}
}

func TestVisitedStore_Concurrent(t *testing.T) {
	vs := NewVisitedStore()
	var wg sync.WaitGroup

	// Concurrent adds
	urls := []string{
		"https://example1.com",
		"https://example2.com",
		"https://example3.com",
		"https://example4.com",
		"https://example5.com",
	}

	for _, url := range urls {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			vs.Add(u)
			vs.Add(u) // duplicate
			vs.Has(u)
		}(url)
	}

	wg.Wait()

	if vs.Count() != len(urls) {
		t.Errorf("Concurrent adds failed, expected %d, got %d", len(urls), vs.Count())
	}
}
