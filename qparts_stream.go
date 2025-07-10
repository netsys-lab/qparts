package qparts

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/netsys-lab/qparts/pkg/qpnet"
)

const (
	/*
	 * Optimize for latency
	 */
	QPARTS_STREAM_PREFERENCE_LATENCY = 1
	/*
	 * Optimize for throughput
	 */
	QPARTS_STREAM_PREFERENCE_THROUGHPUT = 1

	/*
	 * Optimize for fairness, use longer and usually unused paths
	 */
	QPARTS_STREAM_PREFERENCE_FAIRNESS = 1
)

type Conn interface {
	net.Conn
}

var _ Conn = (*PartsStream)(nil)

type PartsStream struct {
	// TODO: Add more fields like preferences, reliable, etc
	Id uint64
	// scheduler  *Scheduler
	ReadBuffer    *qpnet.PacketBuffer
	conn          *QPartsConn
	Preference    uint32
	readDeadline  time.Time
	writeDeadline time.Time
	deadline      time.Time
	deadlineSet   bool       // Tracks if a deadline has been set
	mu            sync.Mutex // Mutex for guarding deadline access
}

func NewPartsStream(id uint64, scheduler *Scheduler) *PartsStream {
	return &PartsStream{
		Id:         id,
		ReadBuffer: qpnet.NewPacketBuffer(),
		// scheduler:  scheduler,
	}
}

func (s *PartsStream) SetPreference(preference uint32) error {

	err := s.conn.ControlPlane.ChangeStreamPreference(s, preference)
	if err != nil {
		return err
	}
	s.Preference = preference
	return nil
}

func (s *PartsStream) Read(b []byte) (n int, err error) {

	// underlying sockets? How to read from many?
	// Assumption: Single socket with multiple streams

	// Copy from s.ReadBuffer to b
	// TODO: Check if there are respective errors in the control plane

	if !s.deadlineSet {
		bts := s.ReadBuffer.Read(len(b))
		n = copy(b, bts)
		return n, nil
	}

	// Deadline is set, lock the mutex to safely access the deadline
	s.mu.Lock()
	readDeadline := s.readDeadline
	s.mu.Unlock()

	// Check for read deadline
	if !readDeadline.IsZero() && time.Now().After(readDeadline) {
		return 0, fmt.Errorf("read timeout")
	}

	bts := s.ReadBuffer.Read(len(b))
	n = copy(b, bts)
	return n, nil
}

func (s *PartsStream) Write(b []byte) (n int, err error) {

	if !s.deadlineSet {
		assignments := s.conn.Dataplane.ScheduleWrite(b, s)

		for i, _ := range assignments.Assignments {
			assignments.Assignments[i].Remote = s.conn.remote
			//Log.Info("----------------------")
			//Log.Info(assignments.Assignments[i].Remote)
		}
		return s.conn.Dataplane.WriteForStream(&assignments, s.Id)
		// return s.conn.Dataplane.WriteDataForStream(b, s.Id)
	}

	// Deadline is set, lock the mutex to safely access the deadline
	s.mu.Lock()
	writeDeadline := s.writeDeadline
	s.mu.Unlock()

	// Check for write deadline
	if !writeDeadline.IsZero() && time.Now().After(writeDeadline) {
		return 0, fmt.Errorf("write timeout")
	}

	assignments := s.conn.Dataplane.ScheduleWrite(b, s)

	for i, _ := range assignments.Assignments {
		assignments.Assignments[i].Remote = s.conn.remote
		//Log.Info("----------------------")
		//Log.Info(assignments.Assignments[i].Remote)
	}
	return s.conn.Dataplane.WriteForStream(&assignments, s.Id)
	// return s.conn.Dataplane.WriteDataForStream(b, s.Id)

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

// SetDeadline sets both the read and write deadlines to the same time.
func (s *PartsStream) SetDeadline(t time.Time) error {
	// Lock only if we need to modify deadlines
	s.mu.Lock()
	defer s.mu.Unlock()

	// Set both read and write deadlines to the same time
	s.deadline = t
	s.readDeadline = t
	s.writeDeadline = t
	s.deadlineSet = true // Mark that a deadline has been set
	return nil
}

// SetReadDeadline sets the deadline for future Read calls.
func (s *PartsStream) SetReadDeadline(t time.Time) error {
	// Lock only if we need to modify deadlines
	s.mu.Lock()
	defer s.mu.Unlock()

	s.readDeadline = t
	s.deadlineSet = true // Mark that a deadline has been set
	return nil
}

// SetWriteDeadline sets the deadline for future Write calls.
func (s *PartsStream) SetWriteDeadline(t time.Time) error {
	// Lock only if we need to modify deadlines
	s.mu.Lock()
	defer s.mu.Unlock()

	s.writeDeadline = t
	s.deadlineSet = true // Mark that a deadline has been set
	return nil
}
