package qparts

import (
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"math/rand"
	"sync"

	"github.com/netsys-lab/qparts/pkg/qpcrypto"
	log "github.com/netsys-lab/qparts/pkg/qplogging"
	"github.com/netsys-lab/qparts/pkg/qpnet"
	"github.com/netsys-lab/qparts/pkg/qpproto"
	"github.com/netsys-lab/qparts/pkg/qpscion"
	"github.com/scionproto/scion/pkg/snet"
)

type ControlPlane struct {
	ErrChan         chan error
	NewStreamChan   chan *PartsStream
	streams         map[uint64]*PartsStream
	local           *snet.UDPAddr
	remote          *snet.UDPAddr
	dp              *QPartsDataplane
	Scheduler       *Scheduler
	ControlConn     *qpnet.SingleStreamQUICConn
	localHandshake  *qpproto.QPartsHandshakePacket
	remoteHandshake *qpproto.QPartsHandshakePacket
	pConn           *QPartsConn
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

func (cp *ControlPlane) Connect(remote *snet.UDPAddr) error {

	paths, err := qpscion.QueryPaths(remote.IA)
	// paths, err := h.queryPaths(context.Background(), remote.IA)
	if err != nil {
		return err
	}

	rAddr := remote.Copy()
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
	hs.StartPortRange = 44000
	hs.EndPortRange = 44010
	hs.Encode()

	cp.localHandshake = hs

	log.Log.Info("Remote CP: ", cp.remote.String())

	n, err := cp.ControlConn.WriteAll(hs.Data)
	if err != nil {
		return err
	}

	log.Log.Info("Sent new stream handshake")
	log.Log.Info("Count ", n)

	// Await reply
	for {
		remoteHs := qpproto.NewQPartsHandshakePacket()
		n, err := cp.ControlConn.ReadAll(remoteHs.Data)
		if err != nil {
			return err
		}

		remoteHs.Decode()
		cp.remoteHandshake = remoteHs
		log.Log.Info("Got reply handshake")
		log.Log.Info("Count ", n)

		cp.RaceDialDataplaneStreams()
		fmt.Println("Done racing")

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
		n, err := cp.ControlConn.ReadAll(remoteHs.Data)
		if err != nil {
			return err
		}

		remoteHs.Decode()
		cp.remoteHandshake = remoteHs

		log.Log.Info("Got incoming handshake")
		log.Log.Info("Count ", n)

		// Send handshake back
		hs := qpproto.NewQPartsHandshakePacket()
		hs.ConnId = newConnId()
		hs.Flags = PARTS_MSG_HS
		hs.Version = getVersion()
		hs.StartPortRange = 42000
		hs.EndPortRange = 42010
		hs.Encode()
		cp.localHandshake = hs

		remote := cp.ControlConn.RemoteAddr()
		cp.remote = remote
		log.Log.Info("Remote CP: ", cp.remote.String())

		go func() {
			n2, err := cp.ControlConn.WriteAll(hs.Data)
			log.Log.Info("Sent reply stream handshake")
			log.Log.Info("Count ", n2)
			if err != nil {
				panic(err)
			}
		}()

		cp.RaceListenDataplaneStreams()
		fmt.Println("Done racing")
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
			panic(err)
		}

		if n <= 0 {
			panic("No data received")
		}

		flags := binary.BigEndian.Uint32(header)
		switch flags {
		case PARTS_MSG_STREAM_HS:
			// Read stream handshake
			p := qpproto.NewQPartsNewStreamPacket()
			n, err := cp.ControlConn.ReadAll(p.Data[4:])
			if err != nil {
				panic(err)
			}

			if n <= 0 {
				panic("No data received")
			}

			copy(p.Data, header)
			p.Decode()

			// TODO: May negotiate stream parameters here
			n2, err := cp.ControlConn.WriteAll(p.Data)
			if err != nil {
				panic(err)
			}

			if n2 <= 0 {
				panic("No data sent")
			}
			// TODO: Send this information over control plane conn?
			s := &PartsStream{
				Id: p.StreamId,
				// scheduler:  cp.Scheduler,
				conn:       cp.pConn,
				ReadBuffer: qpnet.NewPacketBuffer(),
			}
			cp.NewStreamChan <- s
			fmt.Println("Received new stream handshake")

			break
		}
	}
}

func (p *ControlPlane) AcceptStream() (*PartsStream, error) {

	s := <-p.NewStreamChan
	p.streams[s.Id] = s
	fmt.Println("Accepted stream with id ", s.Id)

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
		panic(err)
	}
	if n <= 0 {
		panic("No data sent")
	}

	n2, err := cp.ControlConn.ReadAll(p.Data)
	if err != nil {
		panic(err)
	}
	if n2 <= 0 {
		panic("No data received")
	}

	// TODO: May negotiate stream parameters here
	fmt.Println("Opened stream with id ", s.Id)
	return s, nil
}

func (cp *ControlPlane) RaceDialDataplaneStreams() error {
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(i int) {
			// TODO: Might be a different getCertificateFunc
			ssqc := qpnet.NewSingleStreamQUICConn(getCertificateFunc)
			dpStreamId := newConnId()
			local := cp.local.Copy()
			local.Host.Port = int(cp.localHandshake.StartPortRange) + i
			remote := cp.remote.Copy()
			remote.Host.Port = int(cp.remoteHandshake.StartPortRange) + i

			paths, err := qpscion.QueryPaths(remote.IA)
			// TODO: ErrGroup
			if err != nil {
				panic(err)
			}

			remote.Path = paths[0].Internal.Dataplane()
			err = ssqc.DialAndOpen(local, remote)
			// TODO: ErrGroup
			if err != nil {
				panic(err)
			}

			msg := []byte("Hello from CP")
			_, err = ssqc.WriteAll(msg)
			// TODO: ErrGroup
			if err != nil {
				panic(err)
			}

			_, err = ssqc.ReadAll(msg)
			if err != nil {
				panic(err)
			}

			fmt.Println("Expected: Hello from CP")
			fmt.Println("Received: ", string(msg))

			// TODO: DP Handshake here
			err = cp.dp.AddDialStream(dpStreamId, ssqc, &paths[0])
			// TODO: ErrGroup
			if err != nil {
				panic(err)
			}
			wg.Done()

		}(i)
	}
	wg.Wait()
	fmt.Println("Done waiting")
	go cp.dp.readLoop()
	return nil
}

func (cp *ControlPlane) RaceListenDataplaneStreams() error {
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(i int) {
			// TODO: Might be a different getCertificateFunc
			ssqc := qpnet.NewSingleStreamQUICConn(getCertificateFunc)

			local := cp.local.Copy()
			local.Host.Port = int(cp.localHandshake.StartPortRange) + i

			err := ssqc.ListenAndAccept(local)
			// TODO: ErrGroup
			if err != nil {
				panic(err)
			}

			// TODO: DP Handshake here
			// TODO: Get stream ID here? Do we need it?

			msg := []byte("Hello from CP")
			n, err := ssqc.ReadAll(msg)
			// TODO: ErrGroup
			if err != nil {
				panic(err)
			}
			if n <= 0 {
				panic("No data received")
			}

			_, err = ssqc.WriteAll(msg)
			if err != nil {
				panic(err)
			}

			dpStreamId := newConnId()
			err = cp.dp.AddListenStream(dpStreamId, ssqc)
			// TODO: ErrGroup
			if err != nil {
				panic(err)
			}
			wg.Done()

		}(i)
	}
	wg.Wait()
	fmt.Println("Done waiting")
	go cp.dp.readLoop()
	return nil
}
