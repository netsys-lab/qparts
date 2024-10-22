package qparts

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"time"

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

type GetCertificateFunc func(local string) ([]tls.Certificate, error)

type SingleStreamQUICConn struct {
	local           string
	qconn           quic.Connection
	stream          quic.Stream
	listener        *quic.Listener
	certificateFunc GetCertificateFunc
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

func NewSingleStreamQUICConn(certificateFunc GetCertificateFunc) *SingleStreamQUICConn {
	return &SingleStreamQUICConn{certificateFunc: certificateFunc}
}

func (ssqc *SingleStreamQUICConn) ListenAndAccept(local string) error {
	// local := "1-150,127.0.0.1:4443"
	addr, err := snet.ParseUDPAddr(local)
	if err != nil {
		return err
	}

	lAddr := addr.Host.AddrPort()
	udpAddr := net.UDPAddrFromAddrPort(lAddr)
	conn, err := optimizedconn.Listen(udpAddr)
	if err != nil {
		return err
	}

	pconn := connectedPacketConn{conn}

	certs, err := ssqc.certificateFunc(local)
	if err != nil {
		return err
	}

	listener, err := quic.Listen(pconn, &tls.Config{InsecureSkipVerify: true, Certificates: certs, NextProtos: []string{"qparts"}}, &quic.Config{})
	if err != nil {
		return err
	}

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

func (ssqc *SingleStreamQUICConn) DialAndOpen(local, remote string) error {
	raddr, err := snet.ParseUDPAddr(remote)
	if err != nil {
		return err
	}
	rAddrPort := raddr.Host.AddrPort()
	rudpAddr := net.UDPAddrFromAddrPort(rAddrPort)

	laddr, err := snet.ParseUDPAddr(local)
	if err != nil {
		return err
	}

	lAddrPort := laddr.Host.AddrPort()
	ludpAddr := net.UDPAddrFromAddrPort(lAddrPort)

	h := host()
	paths, err := h.queryPaths(context.Background(), raddr.IA)
	fmt.Println(paths)

	raddr.Path = paths[0].Dataplane()

	conn, err := optimizedconn.Dial(ludpAddr, raddr)
	if err != nil {
		return err
	}

	pconn := connectedPacketConn{conn}

	session, err := quic.Dial(context.Background(), pconn, rudpAddr, &tls.Config{InsecureSkipVerify: true, NextProtos: []string{"qparts"}}, &quic.Config{})
	if err != nil {
		return err
	}

	stream, err := session.OpenStreamSync(context.Background())
	if err != nil {
		return err
	}

	ssqc.qconn = session
	ssqc.stream = stream

	return nil

}
