package qparts

import (
	"sync"

	"github.com/scionproto/scion/pkg/snet"
)

type PartsConn struct {
	sync.Mutex
	local        *snet.UDPAddr
	remote       *snet.UDPAddr
	scheduler    *Scheduler
	Dataplane    *PartsDataplane
	ControlPlane *SbdControlPlane
	Streams      map[uint64]*PartsStream
}

func NewPartsConn(local, remote *snet.UDPAddr) (*PartsConn, error) {
	streams := make(map[uint64]*PartsStream)
	dp := NewDataplane(local, remote, streams)
	Log.Info("Got remote: ", remote.String())
	Log.Info("Got remote Host: ", remote.Host.String())
	partsConn := &PartsConn{
		local:        local,
		remote:       remote,
		Dataplane:    dp,
		ControlPlane: NewControlPlane(local, remote, streams, dp),
		scheduler:    NewScheduler(),
		Streams:      streams,
	}
	partsConn.ControlPlane.Conn = partsConn
	partsConn.ControlPlane.Scheduler = partsConn.scheduler
	partsConn.Dataplane.SetControlPlane(partsConn.ControlPlane)

	// partsConn.scheduler.ActivatePlugin(NewSchedulerRoundRobin())

	go partsConn.ControlPlane.RunCCLoop()
	go partsConn.ControlPlane.RunMetricLoop()

	partsConn.Dataplane.Run()
	return partsConn, nil
}

func (p *PartsConn) AcceptStream() (*PartsStream, error) {

	s, err := p.ControlPlane.AcceptStream()
	if err != nil {
		return nil, err
	}

	p.Streams[s.Id] = s
	// p.Dataplane.streams[s.Id] = s

	return s, nil
}

func (p *PartsConn) OpenStream() (*PartsStream, error) {
	s, err := p.ControlPlane.OpenStream()
	if err != nil {
		return nil, err
	}

	p.Streams[s.Id] = s
	return s, nil
}
