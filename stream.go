package qparts

import (
	"net"
	"time"
)

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
		ReadBuffer: NewPacketBuffer(1024),
		// scheduler:  scheduler,
	}
}

func (s *PartsStream) Read(b []byte) (n int, err error) {

	// underlying sockets? How to read from many?
	// Assumption: Single socket with multiple streams

	// Copy from s.ReadBuffer to b
	// TODO: Check if there are respective errors in the control plane
	var bts []byte
	var success bool
	for !success {
		bts, success = s.ReadBuffer.Dequeue()
	}
	// bts, success = s.ReadBuffer.CopyTo(b)
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
