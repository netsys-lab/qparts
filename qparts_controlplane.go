package qparts

import (
	"crypto/tls"
	"math/rand"

	"github.com/scionproto/scion/pkg/snet"
)

type ControlPlane struct {
	ErrChan     chan error
	streams     map[uint64]*PartsStream
	local       *snet.UDPAddr
	remote      *snet.UDPAddr
	dp          *PartsDataplane
	Scheduler   *Scheduler
	ControlConn *SingleStreamQUICConn
}

func NewQPartsControlPlane(local *snet.UDPAddr, streams map[uint64]*PartsStream) *ControlPlane {
	return &ControlPlane{
		ErrChan:     make(chan error),
		streams:     streams,
		local:       local,
		Scheduler:   NewScheduler(),
		ControlConn: NewSingleStreamQUICConn(getCertificateFunc),
	}
}

// TODO: Move to additional file
func getCertificateFunc(local string) ([]tls.Certificate, error) {
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
	err := cp.ControlConn.DialAndOpen(cp.local.String(), remote.String())
	if err != nil {
		return err
	}

	cp.remote = remote

	// Send handshake
	hs := NewQPartsHandshakePacket()
	hs.ConnId = newConnId()
	hs.Flags = PARTS_MSG_HS
	hs.Version = getVersion()
	hs.StartPortRange = 40000
	hs.EndPortRange = 40010
	hs.Encode()

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

		Log.Info("Got reply handshake")
		Log.Info("Count ", n)
		return nil
	}

}

func (cp *ControlPlane) RemoveStream(id uint64) {
	delete(cp.streams, id)
}

func (cp *ControlPlane) ListenAndAccept() error {
	err := cp.ControlConn.ListenAndAccept(cp.local.String())
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

		Log.Info("Got incoming handshake")
		Log.Info("Count ", n)

		// Send handshake back
		hs := NewQPartsHandshakePacket()
		hs.ConnId = newConnId()
		hs.Flags = PARTS_MSG_HS
		hs.Version = getVersion()
		hs.StartPortRange = 40000
		hs.EndPortRange = 40010
		hs.Encode()

		remote := cp.ControlConn.RemoteAddr()
		cp.remote = remote
		Log.Info("Remote CP: ", cp.remote.String())

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

	// Generate new Parts Stream here

	// p.Streams[s.Id] = s
	// p.Dataplane.streams[s.Id] = s
	s := &PartsStream{}
	return s, nil
}

func (p *ControlPlane) OpenStream() (*PartsStream, error) {
	s := &PartsStream{}
	return s, nil
}
