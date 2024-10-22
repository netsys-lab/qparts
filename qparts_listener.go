package qparts

import (
	"net"

	"github.com/scionproto/scion/pkg/snet"
)

type QPartsListener struct {
	local *snet.UDPAddr
	conn  *QPartsConn
}

func NewQPartsListener(local *snet.UDPAddr) *QPartsListener {
	return &QPartsListener{
		local: local,
	}
}

func (ql *QPartsListener) Accept() (*QPartsConn, error) {

	conn := NewQPartsConn(ql.local)
	ql.conn = conn

	err := conn.ListenAndAccept()
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func (l *QPartsListener) Close() error {
	return l.conn.Close()
}

// Addr returns the local network address that the server is listening on.
func (l *QPartsListener) Addr() net.Addr {
	return l.local
}
