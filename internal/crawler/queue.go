package crawler

import (
	"container/list"
	"sync"
)

// URLItem represents a URL in the queue with its depth
type URLItem struct {
	URL   string
	Depth int
}

// Queue is a thread-safe FIFO queue for URLs
type Queue struct {
	mu    sync.Mutex
	items *list.List
	cond  *sync.Cond
}

// NewQueue creates a new Queue
func NewQueue() *Queue {
	q := &Queue{
		items: list.New(),
	}
	q.cond = sync.NewCond(&q.mu)
	return q
}

// Enqueue adds a URL item to the queue
func (q *Queue) Enqueue(item URLItem) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.items.PushBack(item)
	q.cond.Signal()
}

// EnqueueBatch adds multiple URL items to the queue
func (q *Queue) EnqueueBatch(items []URLItem) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for _, item := range items {
		q.items.PushBack(item)
	}
	q.cond.Broadcast()
}

// Dequeue removes and returns a URL item from the queue
// It blocks if the queue is empty
func (q *Queue) Dequeue() (URLItem, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for q.items.Len() == 0 {
		q.cond.Wait()
	}

	element := q.items.Front()
	if element == nil {
		return URLItem{}, false
	}

	q.items.Remove(element)
	return element.Value.(URLItem), true
}

// TryDequeue attempts to dequeue without blocking
// Returns false if queue is empty
func (q *Queue) TryDequeue() (URLItem, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.items.Len() == 0 {
		return URLItem{}, false
	}

	element := q.items.Front()
	q.items.Remove(element)
	return element.Value.(URLItem), true
}

// Len returns the current length of the queue
func (q *Queue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()

	return q.items.Len()
}

// IsEmpty checks if the queue is empty
func (q *Queue) IsEmpty() bool {
	return q.Len() == 0
}

// Clear removes all items from the queue
func (q *Queue) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.items = list.New()
}