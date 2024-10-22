package qparts

import (
	"sync/atomic"
)

// RingBuffer is a high-performance ring buffer using atomic operations.
type PacketBuffer struct {
	buffer  [][]byte
	size    uint64
	head    uint64
	tail    uint64
	mask    uint64
	entries uint64
}

// NewRingBuffer creates a new RingBuffer with the given size.
func NewPacketBuffer(size uint64) *PacketBuffer {
	// Ensure the size is a power of 2 for masking
	if size&(size-1) != 0 {
		panic("size must be a power of 2")
	}
	return &PacketBuffer{
		buffer: make([][]byte, size),
		size:   size,
		mask:   size - 1,
	}
}

// Enqueue adds an item to the ring buffer.
func (rb *PacketBuffer) Enqueue(item []byte) bool {
	tail := atomic.LoadUint64(&rb.tail)
	head := atomic.LoadUint64(&rb.head)
	if (tail+1)&rb.mask == head&rb.mask {
		// Buffer is full
		return false
	}
	rb.buffer[tail&rb.mask] = item
	atomic.StoreUint64(&rb.tail, tail+1)
	atomic.AddUint64(&rb.entries, 1)
	return true
}

// Dequeue removes an item from the ring buffer.
func (rb *PacketBuffer) Dequeue() ([]byte, bool) {
	head := atomic.LoadUint64(&rb.head)
	tail := atomic.LoadUint64(&rb.tail)
	if head == tail {
		// Buffer is empty
		return nil, false
	}
	item := rb.buffer[head&rb.mask]
	rb.buffer[head&rb.mask] = nil // Avoid memory leak
	atomic.StoreUint64(&rb.head, head+1)
	atomic.AddUint64(&rb.entries, ^uint64(0)) // Subtract 1
	return item, true
}

// Size returns the current number of elements in the ring buffer.
func (rb *PacketBuffer) Size() uint64 {
	return atomic.LoadUint64(&rb.entries)
}

/*
// PacketBuffer is a thread-safe structure for storing packets.
type PacketBuffer struct {
	lock     sync.Mutex
	buffer   [][]byte
	getIndex atomic.Int
	addIndex atomic.Int
	maxSize  int
	// overflow  bool
	emptyChan chan bool
}

// NewPacketBuffer initializes a new PacketBuffer with a given size.
func NewPacketBuffer(size int) *PacketBuffer {
	pb := &PacketBuffer{
		buffer:    make([][]byte, size),
		maxSize:   size,
		emptyChan: make(chan bool),
		addIndex:  0,
		getIndex:  -1,
	}
	return pb
}

// Add adds a new packet to the buffer. If the buffer is full, it removes the oldest packet.
func (pb *PacketBuffer) Add(packet []byte) bool {
	// pb.lock.Lock()
	// defer pb.lock.Unlock()

	if pb.getIndex == -1 {
		select {
		case pb.emptyChan <- true:
			break
		default:
			break
		}

	}

	// getIndex := pb.getIndex.Load()
	addIndex := pb.addIndex.Load()
	if addIndex >= pb.maxSize {
		return false
	}

	// Log.Info("Add to buffer")
	//if len(pb.buffer) >= pb.maxSize {
	//	log.Fatal("Buffer full")
	//	// Remove the oldest packet to make space.
	//	pb.buffer = pb.buffer[1:]
	//}
	pb.buffer[addIndex] = packet
	pb.addIndex.Inc()
}

// Get retrieves and removes the oldest packet from the buffer. Returns false if the buffer is empty.
func (pb *PacketBuffer) Get() ([]byte, bool) {

	getIndex := pb.getIndex.Load()
	addIndex := pb.addIndex.Load()

	if getIndex == -1 {
		<-pb.emptyChan
	}

	//pb.lock.Lock()
	//defer pb.lock.Unlock()
	packet := pb.buffer[pb.getIndex]
	pb.buffer = pb.buffer[1:]
	return packet, true
}

// Size returns the current number of packets in the buffer.
func (pb *PacketBuffer) Size() int {
	pb.lock.Lock()
	defer pb.lock.Unlock()

	return len(pb.buffer)
}

func (pb *PacketBuffer) CopyTo(buf []byte) (int, error) {
	if len(pb.buffer) == 0 {
		<-pb.emptyChan
	}

	pb.lock.Lock()
	defer pb.lock.Unlock()

	count := 0
	index := 0
	// Copy all entries of pb.buffer to buf until buf is full
	for _, packet := range pb.buffer {
		plen := len(packet)
		if index+plen > len(buf) {
			break
		}
		copy(buf[index:plen], packet)
		index += plen
		count++
	}

	// Remove count elements from the front of the buffer
	pb.buffer = pb.buffer[count:]
	Log.Info("Copied ", count, " packets; buffer has ", len(pb.buffer), " packets from ", pb.maxSize)
	return index, nil
}
*/
