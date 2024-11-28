package qparts

import "sync"

type DataSequence struct {
	Data          []byte
	SequenceNo    uint64
	Preference    uint32
	TransferPoint int
}

// Queue represents a thread-safe queue that blocks on Get when empty
type DataQueue struct {
	mu      sync.Mutex
	cond    *sync.Cond
	entries []DataSequence
}

// NewQueue initializes a new Queue
func NewDataQueue() *DataQueue {
	q := &DataQueue{
		entries: []DataSequence{},
	}
	q.cond = sync.NewCond(&q.mu)
	return q
}

// Add adds an item to the queue in a thread-safe manner
func (q *DataQueue) Add(item DataSequence) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.entries = append(q.entries, item)
	q.cond.Signal() // Signal any waiting Get calls
}

// Get retrieves and removes the first item from the queue
// Blocks if the queue is empty until a new item is added
func (q *DataQueue) NextPart(dStream *QPartsDataplaneStream) DataAssignment {
	q.mu.Lock()
	defer q.mu.Unlock()

	for len(q.entries) == 0 {
		q.cond.Wait() // Wait until an item is added
	}

	if dStream.StreamType == QPARTS_DP_STREAM_LOW_LATENCY {
		for i, item := range q.entries {
			// TODO: Another enum value for this?
			if item.Preference == QPARTS_DP_STREAM_LOW_LATENCY {
				q.entries = append(q.entries[:i], q.entries[i+1:]...)
				return DataAssignment{
					Data:       item.Data,
					SequenceId: item.SequenceNo,
				}
			}
		}
	}

	// TODO: Check if this is the last part of the sequence

	//item := q.entries[0]
	q.entries = q.entries[1:] // Remove the item
	return DataAssignment{}
}
