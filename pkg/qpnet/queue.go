package qpnet

import "sync"

// Queue represents a thread-safe queue that blocks on Get when empty
type Queue[T any] struct {
	mu      sync.Mutex
	cond    *sync.Cond
	entries []T
}

// NewQueue initializes a new Queue
func NewQueue[T any]() *Queue[T] {
	q := &Queue[T]{
		entries: []T{},
	}
	q.cond = sync.NewCond(&q.mu)
	return q
}

// Add adds an item to the queue in a thread-safe manner
func (q *Queue[T]) Add(item T) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.entries = append(q.entries, item)
	q.cond.Signal() // Signal any waiting Get calls
}

// Get retrieves and removes the first item from the queue
// Blocks if the queue is empty until a new item is added
func (q *Queue[T]) Get() T {
	q.mu.Lock()
	defer q.mu.Unlock()

	for len(q.entries) == 0 {
		q.cond.Wait() // Wait until an item is added
	}

	item := q.entries[0]
	q.entries = q.entries[1:] // Remove the item
	return item
}
