package qparts

import (
	"sync"

	"github.com/scionproto/scion/pkg/snet"
)

/*
type Connection interface {
	AcceptStream(context.Context) (Stream, error)
	AcceptUniStream(context.Context) (ReceiveStream, error)
	OpenStream() (Stream, error)
	OpenStreamSync(context.Context) (Stream, error)
	OpenUniStream() (SendStream, error)
	OpenUniStreamSync(context.Context) (SendStream, error)
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	CloseWithError(ApplicationErrorCode, string) error
	Context() context.Context
	ConnectionState() ConnectionState
	SendDatagram(payload []byte) error
	ReceiveDatagram(context.Context) ([]byte, error)
}
*/

type QPartsConn struct {
	sync.Mutex
	local        *snet.UDPAddr
	remote       *snet.UDPAddr
	scheduler    *Scheduler
	Dataplane    *QPartsDataplane
	ControlPlane *ControlPlane
	Streams      map[uint64]*PartsStream
}

func NewQPartsConn(local *snet.UDPAddr) *QPartsConn {
	streams := make(map[uint64]*PartsStream)
	scheduler := NewScheduler()
	dp := NewQPartsDataplane(scheduler, streams)

	//Log.Info("Got remote: ", remote.String())
	//Log.Info("Got remote Host: ", remote.Host.String())
	partsConn := &QPartsConn{
		local: local,
		// remote: remote,
		Dataplane: dp,
		scheduler: scheduler,
		Streams:   streams,
	}
	partsConn.ControlPlane = NewQPartsControlPlane(local, streams, dp, partsConn)
	// partsConn.ControlPlane.Conn = partsConn
	partsConn.ControlPlane.Scheduler = partsConn.scheduler
	// partsConn.Dataplane.SetControlPlane(partsConn.ControlPlane)

	// partsConn.scheduler.ActivatePlugin(NewSchedulerRoundRobin())

	// go partsConn.ControlPlane.RunCCLoop()
	// go partsConn.ControlPlane.RunMetricLoop()

	// partsConn.Dataplane.Run()
	return partsConn
}

func (p *QPartsConn) DialAndOpen(remote *snet.UDPAddr) error {
	err := p.ControlPlane.Connect(remote)
	if err != nil {
		return err
	}

	// Open internal QUIC streams
	return nil
}

func (p *QPartsConn) ListenAndAccept() error {
	err := p.ControlPlane.ListenAndAccept()
	if err != nil {
		return err
	}

	// Accept internal QUIC streams
	return nil
}

func (p *QPartsConn) Close() error {
	return p.ControlPlane.ControlConn.Close()
}

func (p *QPartsConn) AcceptStream() (*PartsStream, error) {

	s, err := p.ControlPlane.AcceptStream()
	if err != nil {
		return nil, err
	}

	p.Streams[s.Id] = s
	// p.Dataplane.streams[s.Id] = s

	return s, nil
}

func (p *QPartsConn) OpenStream() (*PartsStream, error) {
	s, err := p.ControlPlane.OpenStream()
	if err != nil {
		return nil, err
	}

	p.Streams[s.Id] = s
	return s, nil
}
