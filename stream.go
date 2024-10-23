package qparts

import (
	"net"
	"sync"
	"time"
)

// PacketBuffer is a structure that holds a buffer of bytes.
// It supports appending bytes to the buffer and reading/removing bytes from it.
type PacketBuffer struct {
	mu     sync.Mutex // protects buffer
	cond   *sync.Cond // condition variable to wait on when buffer is empty
	buffer []byte
}

// NewPacketBuffer creates a new PacketBuffer.
func NewPacketBuffer() *PacketBuffer {
	pb := &PacketBuffer{
		buffer: make([]byte, 0),
	}
	pb.cond = sync.NewCond(&pb.mu)
	return pb
}

// Append adds data to the end of the buffer and signals waiting readers.
func (pb *PacketBuffer) Append(data []byte) {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	// Append the data
	pb.buffer = append(pb.buffer, data...)

	// Signal all waiting goroutines that data has been appended
	pb.cond.Broadcast()
}

// Read removes up to `n` bytes from the beginning of the buffer and returns them.
// If fewer than `n` bytes are available, it returns all available data.
// If the buffer is empty, it will wait until data is available.
func (pb *PacketBuffer) Read(n int) []byte {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	// Wait until the buffer has some data
	for len(pb.buffer) == 0 {
		pb.cond.Wait() // Block until notified that data is available
	}

	// If n is larger than the buffer size, adjust it to read what's available
	if n > len(pb.buffer) {
		n = len(pb.buffer)
	}

	// Read data from the buffer
	data := pb.buffer[:n]
	pb.buffer = pb.buffer[n:] // remove the read data from the buffer
	return data
}

// Size returns the current size of the buffer.
func (pb *PacketBuffer) Size() int {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	return len(pb.buffer)
}

type Conn interface {
	net.Conn
}

var _ Conn = (*PartsStream)(nil)

type PartsStream struct {
	// TODO: Add more fields like preferences, reliable, etc
	Id uint64
	// scheduler  *Scheduler
	ReadBuffer *PacketBuffer
	conn       *QPartsConn
}

func NewPartsStream(id uint64, scheduler *Scheduler) *PartsStream {
	return &PartsStream{
		Id:         id,
		ReadBuffer: NewPacketBuffer(),
		// scheduler:  scheduler,
	}
}

func (s *PartsStream) Read(b []byte) (n int, err error) {

	// underlying sockets? How to read from many?
	// Assumption: Single socket with multiple streams

	// Copy from s.ReadBuffer to b
	// TODO: Check if there are respective errors in the control plane

	bts := s.ReadBuffer.Read(len(b))
	n = copy(b, bts)
	return n, nil
}

func (s *PartsStream) Write(b []byte) (n int, err error) {

	assignments := s.conn.Dataplane.ScheduleWrite(b, s)

	for i, _ := range assignments.Assignments {
		assignments.Assignments[i].Remote = s.conn.remote
		//Log.Info("----------------------")
		//Log.Info(assignments.Assignments[i].Remote)
	}
	return s.conn.Dataplane.WriteForStream(&assignments, s.Id)

}

// TODO: Implement
func (s *PartsStream) Close() error {
	// s.Conn.ControlPlane.RemoveStream(s.Id)
	// s.Conn.Dataplane.RemoveStream(s.Id)
	return nil
}

func (s *PartsStream) LocalAddr() net.Addr {
	return s.conn.local
}

func (s *PartsStream) RemoteAddr() net.Addr {
	return s.conn.remote
}

// TODO: Implement these, maybe by a single mutex or something
func (s *PartsStream) SetDeadline(t time.Time) error {
	return nil
}

func (s *PartsStream) SetReadDeadline(t time.Time) error {
	return nil
}

func (s *PartsStream) SetWriteDeadline(t time.Time) error {
	return nil
}
