package qpnet

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net"
	"time"

	"github.com/netsys-lab/qparts/pkg/qpcrypto"
	"github.com/netsys-lab/qparts/pkg/qpscion"
	optimizedconn "github.com/netsys-lab/scion-optimized-connection/pkg"

	"github.com/quic-go/quic-go"
	"github.com/scionproto/scion/pkg/snet"
)

type connectedPacketConn struct {
	net.Conn
}

func (c connectedPacketConn) WriteTo(b []byte, to net.Addr) (int, error) {
	return c.Write(b)
}

func (c connectedPacketConn) ReadFrom(b []byte) (int, net.Addr, error) {
	n, err := c.Read(b)
	return n, c.RemoteAddr(), err
}

type GetCertificateFunc func(local *snet.UDPAddr) ([]tls.Certificate, error)

type SingleStreamQUICConn struct {
	local           *snet.UDPAddr
	qconn           quic.Connection
	stream          quic.Stream
	listener        *quic.Listener
	certificateFunc GetCertificateFunc
	optConn         *optimizedconn.OptimizedSCIONConn
	QTracer         *QPartsQuicTracer
	Path            *qpscion.QPartsPath
}

func (ssqc *SingleStreamQUICConn) Close() error {
	return ssqc.stream.Close()
}

func (ssqc *SingleStreamQUICConn) Context() context.Context {
	return ssqc.stream.Context()
}
func (ssqc *SingleStreamQUICConn) Read(p []byte) (n int, err error) {
	return ssqc.stream.Read(p)
}
func (ssqc *SingleStreamQUICConn) ReadAll(p []byte) (n int, err error) {
	return io.ReadFull(ssqc.stream, p)

}
func (ssqc *SingleStreamQUICConn) SetReadDeadline(t time.Time) error {
	return ssqc.stream.SetReadDeadline(t)
}
func (ssqc *SingleStreamQUICConn) SetWriteDeadline(t time.Time) error {
	return ssqc.stream.SetWriteDeadline(t)
}
func (ssqc *SingleStreamQUICConn) WriteAll(p []byte) (n int64, err error) {
	return io.CopyN(ssqc.stream, bytes.NewReader(p), int64(len(p)))
}
func (ssqc *SingleStreamQUICConn) Write(p []byte) (n int, err error) {
	return ssqc.stream.Write(p)
}

func (ssqc *SingleStreamQUICConn) RemoteAddr() *snet.UDPAddr {
	return ssqc.qconn.RemoteAddr().(*snet.UDPAddr)
}

// TODO: Set path in optimizedconn
func (ssqc *SingleStreamQUICConn) SetPath(path *qpscion.QPartsPath) error {
	// err := ssqc.optConn.SetPath(path)
	// return err
	ssqc.Path = path
	return nil
}

func NewSingleStreamQUICConn(certificateFunc GetCertificateFunc) *SingleStreamQUICConn {
	return &SingleStreamQUICConn{certificateFunc: certificateFunc, QTracer: NewQPartsQuicTracer()}
}

func (ssqc *SingleStreamQUICConn) ListenAndAccept(local *snet.UDPAddr) error {
	ssqc.local = local
	udpAddr := net.UDPAddrFromAddrPort(local.Host.AddrPort())
	conn, err := optimizedconn.Listen(udpAddr)
	if err != nil {
		return err
	}

	pconn := connectedPacketConn{conn}

	/*certs, err := ssqc.certificateFunc(local)
	if err != nil {
		return err
	}*/

	tlsConf, err := qpcrypto.NewQPartsListenTLSConfig(local)
	if err != nil {
		return err
	}

	silenceLog()
	listener, err := quic.Listen(pconn, tlsConf, &quic.Config{Tracer: ssqc.QTracer.NewTracerHandler(), DisablePathMTUDiscovery: true, KeepAlivePeriod: 1 * time.Second})
	if err != nil {
		return err
	}
	unsilenceLog()

	for {
		sess, err := listener.Accept(context.Background())
		if err != nil {
			return err
		}

		stream, err := sess.AcceptStream(context.Background())
		if err != nil {
			return err
		}

		ssqc.qconn = sess
		ssqc.stream = stream
		ssqc.listener = listener

		break
	}

	return nil
}

func (ssqc *SingleStreamQUICConn) DialAndOpen(local, remote *snet.UDPAddr) error {
	rAddrPort := remote.Host.AddrPort()
	rudpAddr := net.UDPAddrFromAddrPort(rAddrPort)

	lAddrPort := local.Host.AddrPort()
	ludpAddr := net.UDPAddrFromAddrPort(lAddrPort)

	ssqc.local = local

	conn, err := optimizedconn.Dial(ludpAddr, remote)
	if err != nil {
		return err
	}

	pconn := connectedPacketConn{conn}

	tlsConf := qpcrypto.NewQPartsDialTLSConfig(remote)
	silenceLog()
	session, err := quic.Dial(context.Background(), pconn, rudpAddr, tlsConf, &quic.Config{Tracer: ssqc.QTracer.NewTracerHandler(), DisablePathMTUDiscovery: true, KeepAlivePeriod: 1 * time.Second})
	if err != nil {
		return err
	}
	unsilenceLog()

	stream, err := session.OpenStreamSync(context.Background())
	if err != nil {
		return err
	}

	ssqc.qconn = session
	ssqc.stream = stream

	return nil

}
