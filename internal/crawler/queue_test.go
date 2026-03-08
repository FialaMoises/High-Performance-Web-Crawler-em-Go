package crawler

import (
	"sync"
	"testing"
)

func TestQueue_EnqueueDequeue(t *testing.T) {
	q := NewQueue()

	item := URLItem{URL: "https://example.com", Depth: 0}
	q.Enqueue(item)

	dequeued, ok := q.TryDequeue()
	if !ok {
		t.Fatal("TryDequeue should return true when queue has items")
	}

	if dequeued.URL != item.URL || dequeued.Depth != item.Depth {
		t.Error("Dequeued item doesn't match enqueued item")
	}
}

func TestQueue_TryDequeue_Empty(t *testing.T) {
	q := NewQueue()

	_, ok := q.TryDequeue()
	if ok {
		t.Error("TryDequeue should return false when queue is empty")
	}
}

func TestQueue_EnqueueBatch(t *testing.T) {
	q := NewQueue()

	items := []URLItem{
		{URL: "https://example1.com", Depth: 0},
		{URL: "https://example2.com", Depth: 1},
		{URL: "https://example3.com", Depth: 2},
	}

	q.EnqueueBatch(items)

	if q.Len() != 3 {
		t.Errorf("Queue length should be 3, got %d", q.Len())
	}

	// Dequeue all items
	for i := 0; i < 3; i++ {
		item, ok := q.TryDequeue()
		if !ok {
			t.Fatalf("Failed to dequeue item %d", i)
		}
		if item.URL != items[i].URL {
			t.Errorf("Item %d URL mismatch: expected %s, got %s", i, items[i].URL, item.URL)
		}
	}
}

func TestQueue_Len(t *testing.T) {
	q := NewQueue()

	if q.Len() != 0 {
		t.Errorf("Initial length should be 0, got %d", q.Len())
	}

	q.Enqueue(URLItem{URL: "https://example1.com", Depth: 0})
	q.Enqueue(URLItem{URL: "https://example2.com", Depth: 1})

	if q.Len() != 2 {
		t.Errorf("Length should be 2, got %d", q.Len())
	}

	q.TryDequeue()

	if q.Len() != 1 {
		t.Errorf("Length should be 1 after dequeue, got %d", q.Len())
	}
}

func TestQueue_IsEmpty(t *testing.T) {
	q := NewQueue()

	if !q.IsEmpty() {
		t.Error("New queue should be empty")
	}

	q.Enqueue(URLItem{URL: "https://example.com", Depth: 0})

	if q.IsEmpty() {
		t.Error("Queue should not be empty after enqueue")
	}

	q.TryDequeue()

	if !q.IsEmpty() {
		t.Error("Queue should be empty after dequeuing all items")
	}
}

func TestQueue_Clear(t *testing.T) {
	q := NewQueue()

	q.Enqueue(URLItem{URL: "https://example1.com", Depth: 0})
	q.Enqueue(URLItem{URL: "https://example2.com", Depth: 1})

	q.Clear()

	if !q.IsEmpty() {
		t.Error("Queue should be empty after Clear")
	}

	if q.Len() != 0 {
		t.Errorf("Queue length should be 0 after Clear, got %d", q.Len())
	}
}

func TestQueue_Concurrent(t *testing.T) {
	q := NewQueue()
	var wg sync.WaitGroup

	// Concurrent enqueues
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			q.Enqueue(URLItem{
				URL:   "https://example.com/" + string(rune(idx)),
				Depth: idx,
			})
		}(i)
	}

	wg.Wait()

	if q.Len() != 100 {
		t.Errorf("Expected 100 items in queue, got %d", q.Len())
	}

	// Concurrent dequeues
	count := 0
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, ok := q.TryDequeue(); ok {
				count++
			}
		}()
	}

	wg.Wait()

	if count != 100 {
		t.Errorf("Expected to dequeue 100 items, got %d", count)
	}

	if !q.IsEmpty() {
		t.Error("Queue should be empty after dequeuing all items")
	}
}

func TestQueue_FIFO(t *testing.T) {
	q := NewQueue()

	// Enqueue in order
	items := []URLItem{
		{URL: "first", Depth: 0},
		{URL: "second", Depth: 1},
		{URL: "third", Depth: 2},
	}

	for _, item := range items {
		q.Enqueue(item)
	}

	// Dequeue should maintain order (FIFO)
	for i, expected := range items {
		dequeued, ok := q.TryDequeue()
		if !ok {
			t.Fatalf("Failed to dequeue item %d", i)
		}
		if dequeued.URL != expected.URL {
			t.Errorf("FIFO order violated: expected %s, got %s", expected.URL, dequeued.URL)
		}
	}
}
