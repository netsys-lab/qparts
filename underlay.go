package qparts

import (
	"net"
	"net/netip"
	"time"

	"github.com/scionproto/scion/pkg/snet"
)

const (
	MAX_MTU = 8952
)

type SCIONPacket struct {
	Payload []byte
	Path    snet.Path
	Addr    *snet.UDPAddr
}

type UnderlaySCIONSocket interface {
	ReadMany() ([]*SCIONPacket, error)
	WriteMany([]*SCIONPacket) (int, error)
	Listen(local netip.AddrPort) error
	Close() error
	Conn() net.PacketConn
	SetDeadline(t time.Time) error
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
}

type UnderlaySocket interface {
	ReadMany() ([]*SCIONSocketPacket, error)
	WriteMany([]*SCIONSocketPacket) (int, error)
	Listen(local netip.AddrPort) error
	Close() error
	Conn() net.PacketConn
	SetDeadline(t time.Time) error
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
}
