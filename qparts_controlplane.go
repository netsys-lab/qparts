package qparts

import (
	"context"
	"crypto/tls"
	"fmt"
	"math/rand"
	"sync"

	"github.com/scionproto/scion/pkg/snet"
)

type ControlPlane struct {
	ErrChan         chan error
	streams         map[uint64]*PartsStream
	local           *snet.UDPAddr
	remote          *snet.UDPAddr
	dp              *QPartsDataplane
	Scheduler       *Scheduler
	ControlConn     *SingleStreamQUICConn
	localHandshake  *QPartsHandshakePacket
	remoteHandshake *QPartsHandshakePacket
	pConn           *QPartsConn
}

func NewQPartsControlPlane(local *snet.UDPAddr, streams map[uint64]*PartsStream, dp *QPartsDataplane, pconn *QPartsConn) *ControlPlane {
	return &ControlPlane{
		ErrChan:     make(chan error),
		streams:     streams,
		local:       local,
		dp:          dp,
		pConn:       pconn,
		Scheduler:   NewScheduler(),
		ControlConn: NewSingleStreamQUICConn(getCertificateFunc),
	}
}

// TODO: Move to additional file
func getCertificateFunc(local *snet.UDPAddr) ([]tls.Certificate, error) {
	certs := MustGenerateSelfSignedCert()
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

	h := host()
	paths, err := h.queryPaths(context.Background(), remote.IA)
	if err != nil {
		return err
	}

	rAddr := remote.Copy()
	rAddr.Path = paths[0].Dataplane()

	err = cp.ControlConn.DialAndOpen(cp.local, rAddr)
	if err != nil {
		return err
	}

	cp.remote = remote

	// Send handshake
	hs := NewQPartsHandshakePacket()
	hs.ConnId = newConnId()
	hs.Flags = PARTS_MSG_HS
	hs.Version = getVersion()
	hs.StartPortRange = 44000
	hs.EndPortRange = 44010
	hs.Encode()

	cp.localHandshake = hs

	Log.Info("Remote CP: ", cp.remote.String())

	n, err := cp.ControlConn.WriteAll(hs.Data)
	if err != nil {
		return err
	}

	Log.Info("Sent new stream handshake")
	Log.Info("Count ", n)
	if err != nil {
		return err
	}

	// Await reply
	for {
		remoteHs := NewQPartsHandshakePacket()
		n, err := cp.ControlConn.ReadAll(remoteHs.Data)
		if err != nil {
			return err
		}

		remoteHs.Decode()
		cp.remoteHandshake = remoteHs
		Log.Info("Got reply handshake")
		Log.Info("Count ", n)

		go func() {
			cp.RaceDialDataplaneStreams()
		}()

		return nil
	}

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
		remoteHs := NewQPartsHandshakePacket()
		n, err := cp.ControlConn.ReadAll(remoteHs.Data)
		if err != nil {
			return err
		}

		remoteHs.Decode()
		cp.remoteHandshake = remoteHs

		Log.Info("Got incoming handshake")
		Log.Info("Count ", n)

		// Send handshake back
		hs := NewQPartsHandshakePacket()
		hs.ConnId = newConnId()
		hs.Flags = PARTS_MSG_HS
		hs.Version = getVersion()
		hs.StartPortRange = 42000
		hs.EndPortRange = 42010
		hs.Encode()
		cp.localHandshake = hs

		remote := cp.ControlConn.RemoteAddr()
		cp.remote = remote
		Log.Info("Remote CP: ", cp.remote.String())

		go func() {
			cp.RaceListenDataplaneStreams()
		}()

		n2, err := cp.ControlConn.WriteAll(hs.Data)
		if err != nil {
			return err
		}

		Log.Info("Sent reply stream handshake")
		Log.Info("Count ", n2)
		break
	}
	return nil
}

func (p *ControlPlane) AcceptStream() (*PartsStream, error) {

	// TODO: Send this information over control plane conn?
	s := &PartsStream{
		Id:         newConnId(),
		scheduler:  p.Scheduler,
		conn:       p.pConn,
		ReadBuffer: NewPacketBuffer(1024),
	}
	p.streams[s.Id] = s

	return s, nil
}

func (p *ControlPlane) OpenStream() (*PartsStream, error) {
	s := &PartsStream{
		Id:         newConnId(),
		scheduler:  p.Scheduler,
		conn:       p.pConn,
		ReadBuffer: NewPacketBuffer(1024),
	}
	p.streams[s.Id] = s
	return s, nil
}

func (cp *ControlPlane) RaceDialDataplaneStreams() error {
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(i int) {
			// TODO: Might be a different getCertificateFunc
			ssqc := NewSingleStreamQUICConn(getCertificateFunc)
			dpStreamId := newConnId()
			local := cp.local.Copy()
			local.Host.Port = int(cp.localHandshake.StartPortRange) + i
			remote := cp.remote.Copy()
			remote.Host.Port = int(cp.remoteHandshake.StartPortRange) + i

			h := host()
			paths, err := h.queryPaths(context.Background(), remote.IA)
			// TODO: ErrGroup
			if err != nil {
				panic(err)
			}

			remote.Path = paths[0].Dataplane()
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
			err = cp.dp.AddDialStream(dpStreamId, ssqc, remote.Path)
			// TODO: ErrGroup
			if err != nil {
				panic(err)
			}
			wg.Done()

		}(i)
	}
	wg.Wait()
	fmt.Println("Done waiting")
	cp.dp.readLoop()
	return nil
}

func (cp *ControlPlane) RaceListenDataplaneStreams() error {
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(i int) {
			// TODO: Might be a different getCertificateFunc
			ssqc := NewSingleStreamQUICConn(getCertificateFunc)

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
	cp.dp.readLoop()
	return nil
}
