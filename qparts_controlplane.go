package qparts

import (
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"math/rand"
	"sync"

	"github.com/netsys-lab/qparts/pkg/qpcrypto"
	"github.com/netsys-lab/qparts/pkg/qplogging"
	log "github.com/netsys-lab/qparts/pkg/qplogging"
	"github.com/netsys-lab/qparts/pkg/qpmetrics"
	"github.com/netsys-lab/qparts/pkg/qpnet"
	"github.com/netsys-lab/qparts/pkg/qpproto"
	"github.com/netsys-lab/qparts/pkg/qpscion"
	"github.com/scionproto/scion/pkg/snet"
)

type ControlPlane struct {
	ErrChan              chan error
	NewStreamChan        chan *PartsStream
	streams              map[uint64]*PartsStream
	local                *snet.UDPAddr
	remote               *snet.UDPAddr
	dp                   *QPartsDataplane
	Scheduler            *Scheduler
	ControlConn          *qpnet.SingleStreamQUICConn
	localHandshake       *qpproto.QPartsHandshakePacket
	remoteHandshake      *qpproto.QPartsHandshakePacket
	pConn                *QPartsConn
	initialPathSelection []qpscion.QPartsPath
}

func NewQPartsControlPlane(local *snet.UDPAddr, streams map[uint64]*PartsStream, dp *QPartsDataplane, pconn *QPartsConn) *ControlPlane {
	return &ControlPlane{
		ErrChan:       make(chan error),
		streams:       streams,
		local:         local,
		dp:            dp,
		NewStreamChan: make(chan *PartsStream),
		pConn:         pconn,
		Scheduler:     NewScheduler(),
		ControlConn:   qpnet.NewSingleStreamQUICConn(getCertificateFunc),
	}
}

// TODO: Move to additional file
func getCertificateFunc(local *snet.UDPAddr) ([]tls.Certificate, error) {
	certs := qpcrypto.MustGenerateSelfSignedCert()
	return certs, nil
}

func newConnId() uint64 {
	// Generate Random number
	return rand.Uint64()
}

func getVersion() uint64 {
	return 1
}

func (cp *ControlPlane) HandleCongestionEvent(event *qpmetrics.CongestionEvent) error {
	return nil
}

func (cp *ControlPlane) Connect(remote *snet.UDPAddr) error {

	paths, err := qpscion.Paths.Get(remote)
	if err != nil {
		return err
	}

	// TODO: Conn Preference?
	initialPathSelection, err := cp.Scheduler.InitialPathSelection(0, paths)
	if err != nil {
		return err
	}

	cp.initialPathSelection = initialPathSelection

	rAddr := remote.Copy()
	rAddr.NextHop = paths[0].Internal.UnderlayNextHop()
	rAddr.Path = paths[0].Internal.Dataplane()

	err = cp.ControlConn.DialAndOpen(cp.local, rAddr)
	if err != nil {
		return err
	}

	cp.remote = remote

	// Send handshake
	hs := qpproto.NewQPartsHandshakePacket()
	hs.ConnId = newConnId()
	hs.Flags = PARTS_MSG_HS
	hs.Version = getVersion()
	hs.StartPortRange = 31500
	hs.EndPortRange = 31510
	hs.NumStreams = uint16(len(initialPathSelection))
	hs.Encode()

	cp.localHandshake = hs

	log.Log.Debug("Remote CP: ", cp.remote.String())

	_, err = cp.ControlConn.WriteAll(hs.Data)
	if err != nil {
		return err
	}

	log.Log.Debug("Sent new stream handshake")
	// log.Log.Info("Count ", n)

	// Await reply
	for {
		remoteHs := qpproto.NewQPartsHandshakePacket()
		_, err := cp.ControlConn.ReadAll(remoteHs.Data)
		if err != nil {
			return err
		}

		remoteHs.Decode()
		cp.remoteHandshake = remoteHs
		log.Log.Debug("Got reply handshake")
		// log.Log.Info("Count ", n)

		cp.RaceDialDataplaneStreams()
		qplogging.Log.Debug("Done racing")

		break
	}

	go cp.readLoop()
	return nil

}

func (cp *ControlPlane) RemoveStream(id uint64) {
	delete(cp.streams, id)
}

func (cp *ControlPlane) ListenAndAccept() error {
	err := cp.ControlConn.ListenAndAccept(cp.local)
	if err != nil {
		return err
	}

	// Receive handshake
	for {
		remoteHs := qpproto.NewQPartsHandshakePacket()
		_, err := cp.ControlConn.ReadAll(remoteHs.Data)
		if err != nil {
			return err
		}

		remoteHs.Decode()

		// TODO: Check version, nuance whatever to ensure this is a valid packet

		cp.remoteHandshake = remoteHs

		log.Log.Debug("Got incoming handshake")
		// log.Log.Info("Count ", n)

		// Send handshake back
		hs := qpproto.NewQPartsHandshakePacket()
		hs.ConnId = newConnId()
		hs.Flags = PARTS_MSG_HS
		hs.Version = getVersion()
		hs.StartPortRange = 31600
		hs.EndPortRange = 31610
		hs.Encode()
		cp.localHandshake = hs

		remote := cp.ControlConn.RemoteAddr()
		cp.remote = remote
		log.Log.Debug("Remote CP: ", cp.remote.String())

		//go func() {
		_, err = cp.ControlConn.WriteAll(hs.Data)

		// log.Log.Info("Count ", n2)
		if err != nil {
			return err
		}
		log.Log.Debug("Sent reply stream handshake")
		//}()

		cp.RaceListenDataplaneStreams(int(remoteHs.NumStreams))
		qplogging.Log.Debug("Done racing")
		break
	}
	go cp.readLoop()
	return nil
}

func (cp *ControlPlane) readLoop() error {
	for {
		header := make([]byte, 4)
		n, err := cp.ControlConn.ReadAll(header)
		if err != nil {
			qplogging.Log.Warn("Failed to read from control conn: ", err)
		}

		if n <= 0 {
			qplogging.Log.Debug("No data received on control conn")
			continue
		}

		flags := binary.BigEndian.Uint32(header)
		switch flags {
		case PARTS_MSG_STREAM_HS:
			// Read stream handshake
			p := qpproto.NewQPartsNewStreamPacket()
			n, err := cp.ControlConn.ReadAll(p.Data[4:])
			if err != nil {
				qplogging.Log.Warn("Failed to parse PARTS_MSG_STREAM_HS: ", err)
			}

			if n <= 0 {
				qplogging.Log.Debug("Failed to read PARTS_MSG_STREAM_HS data from control conn")
				continue
			}

			copy(p.Data, header)
			p.Decode()

			// TODO: May negotiate stream parameters here
			n2, err := cp.ControlConn.WriteAll(p.Data)
			if err != nil {
				qplogging.Log.Warn("Failed to write to control conn: ", err)
				continue
			}

			if n2 <= 0 {
				qplogging.Log.Debug("Failed to write PARTS_MSG_STREAM_HS data to control conn")
				continue
			}
			// TODO: Send this information over control plane conn?
			s := &PartsStream{
				Id:         p.StreamId,
				Preference: p.StreamProperty,
				conn:       cp.pConn,
				ReadBuffer: qpnet.NewPacketBuffer(),
			}
			qplogging.Log.Info("Got new PartsStream ", p.StreamId)
			cp.NewStreamChan <- s
			qplogging.Log.Debug("Received new stream handshake")
			break
		case PARTS_MSG_STREAM_PROPERTY:
			// Read stream handshake
			p := qpproto.NewQPartsNewStreamPacket()
			n, err := cp.ControlConn.ReadAll(p.Data[4:])
			if err != nil {
				qplogging.Log.Warn("Failed to read PARTS_MSG_STREAM_PROPERTY from control conn: ", err)
				continue
			}

			if n <= 0 {
				qplogging.Log.Debug("Failed to read PARTS_MSG_STREAM_PROPERTY data to control conn")
				continue
			}

			copy(p.Data, header)
			p.Decode()

			// TODO: May negotiate stream parameters here
			n2, err := cp.ControlConn.WriteAll(p.Data)
			if err != nil {
				qplogging.Log.Warn("Failed to write to control conn: ", err)
				continue
			}

			if n2 <= 0 {
				qplogging.Log.Debug("Failed to write PARTS_MSG_STREAM_PROPERTY data to control conn")
				continue
			}
			cp.streams[p.StreamId].Preference = p.StreamProperty
			qplogging.Log.Debug("Update stream preference")

			break
		}
	}
}

func (p *ControlPlane) AcceptStream() (*PartsStream, error) {

	s := <-p.NewStreamChan
	p.streams[s.Id] = s
	qplogging.Log.Debug("Accepted stream with id ", s.Id)

	return s, nil
}

func (cp *ControlPlane) OpenStream() (*PartsStream, error) {
	s := &PartsStream{
		Id: newConnId(),
		// scheduler:  cp.Scheduler,
		conn:       cp.pConn,
		ReadBuffer: qpnet.NewPacketBuffer(),
	}

	p := qpproto.NewQPartsNewStreamPacket()
	p.StreamId = s.Id
	p.Flags = PARTS_MSG_STREAM_HS
	p.Encode()

	n, err := cp.ControlConn.WriteAll(p.Data)
	if err != nil {
		return nil, err
	}
	if n <= 0 {
		return nil, fmt.Errorf("no data sent to control conn to open stream")
	}

	n2, err := cp.ControlConn.ReadAll(p.Data)
	if err != nil {
		return nil, err
	}
	if n2 <= 0 {
		return nil, fmt.Errorf("no data received from control conn to open stream")
	}

	// TODO: May negotiate stream parameters here
	qplogging.Log.Info("Got new PartsStream ", p.StreamId)
	qplogging.Log.Debug("Opened stream with id ", s.Id)
	return s, nil
}

func (cp *ControlPlane) ChangeStreamPreference(s *PartsStream, pref uint32) error {
	p := qpproto.NewQPartsNewStreamPacket()
	p.StreamId = s.Id
	p.Flags = PARTS_MSG_STREAM_PROPERTY
	p.StreamProperty = pref
	p.Encode()

	n, err := cp.ControlConn.WriteAll(p.Data)
	if err != nil {
		return err
	}
	if n <= 0 {
		return fmt.Errorf("no data sent to control conn to change stream preference")
	}

	n2, err := cp.ControlConn.ReadAll(p.Data)
	if err != nil {
		return err
	}
	if n2 <= 0 {
		return fmt.Errorf("no data received from control conn to change stream preference")
	}

	// TODO: May negotiate stream parameters here
	qplogging.Log.Debug("Changed stream preference with id ", s.Id)
	return nil
}

func (cp *ControlPlane) RaceDialDataplaneStreams() error {
	var wg sync.WaitGroup
	errStr := ""
	qplogging.Log.Debug("Racing streams over ", len(cp.initialPathSelection), " paths")
	for i, path := range cp.initialPathSelection {
		go func(i int, path *qpscion.QPartsPath) {
			// TODO: Might be a different getCertificateFunc
			defer wg.Done()
			ssqc := qpnet.NewSingleStreamQUICConn(getCertificateFunc)
			dpStreamId := newConnId()
			local := cp.local.Copy()
			local.Host.Port = int(cp.localHandshake.StartPortRange) + i
			remote := cp.remote.Copy()
			remote.Host.Port = int(cp.remoteHandshake.StartPortRange) + i

			remote.Path = path.Internal.Dataplane()
			remote.NextHop = path.Internal.UnderlayNextHop()
			err := ssqc.DialAndOpen(local, remote)
			// TODO: ErrGroup
			if err != nil {
				errStr += err.Error()
				return
			}

			ssqc.SetPath(path)

			msg := []byte("Hello from CP")
			_, err = ssqc.WriteAll(msg)
			// TODO: ErrGroup
			if err != nil {
				errStr += err.Error()
				return
			}

			_, err = ssqc.ReadAll(msg)
			if err != nil {
				errStr += err.Error()
				return
			}

			qplogging.Log.Debug("Expected: Hello from CP")
			qplogging.Log.Debug("Received: ", string(msg))

			// TODO: DP Handshake here
			err = cp.dp.AddDialStream(dpStreamId, ssqc, path)
			// TODO: ErrGroup
			if err != nil {
				errStr += err.Error()
				return
			}

		}(i, &path)
	}
	wg.Wait()

	if errStr != "" {
		return fmt.Errorf("failed to dial at least one stream %s", errStr)
	}

	qplogging.Log.Debug("Done waiting")
	go cp.dp.readLoop()
	go cp.dp.writeLoop()
	return nil
}

func (cp *ControlPlane) RaceListenDataplaneStreams(numStreams int) error {
	var wg sync.WaitGroup
	qplogging.Log.Debug("Racing streams over ", numStreams, " paths")
	var errStr string
	for i := 0; i < numStreams; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			// TODO: Might be a different getCertificateFunc
			ssqc := qpnet.NewSingleStreamQUICConn(getCertificateFunc)

			local := cp.local.Copy()
			local.Host.Port = int(cp.localHandshake.StartPortRange) + i

			err := ssqc.ListenAndAccept(local)
			// TODO: ErrGroup
			if err != nil {
				errStr += err.Error()
				return
			}
			// TODO: DP Handshake here
			// TODO: Get stream ID here? Do we need it?

			msg := []byte("Hello from CP")
			n, err := ssqc.ReadAll(msg)
			// TODO: ErrGroup
			if err != nil {
				errStr += err.Error()
				return
			}
			if n <= 0 {
				errStr += "No data received"
				return
			}

			_, err = ssqc.WriteAll(msg)
			if err != nil {
				errStr += err.Error()
				return
			}

			dpStreamId := newConnId()
			err = cp.dp.AddListenStream(dpStreamId, ssqc)
			// TODO: ErrGroup
			if err != nil {
				errStr += err.Error()
				return
			}

		}(i)
	}
	wg.Wait()

	if errStr != "" {
		return fmt.Errorf("failed to listen at least one stream %s", errStr)
	}

	qplogging.Log.Debug("Done waiting")
	go cp.dp.readLoop()
	go cp.dp.writeLoop()
	return nil
}
