package qparts

import (
	"net"
	"net/netip"
	"time"

	optimizedconn "github.com/netsys-lab/scion-optimized-connection/pkg"
)

type SCIONOptimizedSocket struct {
	local   string
	remote  string
	conn    *optimizedconn.OptimizedSCIONPacketConn
	counter int
}

func NewSCIONOptimizedSocket() *SCIONOptimizedSocket {
	return &SCIONOptimizedSocket{}

}

// TODO: Could be recvmany
func (ps *SCIONOptimizedSocket) ReadMany() ([]*SCIONPacket, error) {
	buf := make([]byte, MAX_MTU)
	n, _, err := ps.conn.ReadFrom(buf)
	if err != nil {
		return nil, err
	}

	sp := SCIONPacket{
		Payload: buf[:n],
	}

	//if remote != nil {
	//	sp.Addr = remote
	//}

	return []*SCIONPacket{&sp}, nil

}

func (ps *SCIONOptimizedSocket) WriteMany(pkts []*SCIONPacket) (int, error) {
	sentPackets := 0
	for _, v := range pkts {

		if v.Addr.Path == nil {
			v.Addr.Path = v.Path.Dataplane()
			v.Addr.NextHop = v.Path.UnderlayNextHop()
		}

		_, err := ps.conn.WriteTo(v.Payload, v.Addr)
		if err != nil {
			Log.Info("Failed to write Packet ", err)
			return sentPackets, err
		}
		ps.counter++

		/*if ps.counter > 100 {
			Log.Info("adsklasökd GO EHAED")
			for {
				_, err := ps.conn.WriteTo(v.Payload, v.Addr)
				if err != nil {
					Log.Info(err)
					return sentPackets, err
				}

			}
		}*/

		sentPackets++
	}
	// Log.Info("Sent packets: ", sentPackets)
	return sentPackets, nil
}

func (ps *SCIONOptimizedSocket) Listen(local netip.AddrPort) error {

	// ps.local = local
	udpAddr := net.UDPAddrFromAddrPort(local)
	//conn, err := net.ListenUDP("udp", udpAddr)
	//if err != nil {
	//	return err
	//}

	conn, err := optimizedconn.ListenPacket(udpAddr)
	if err != nil {
		return err
	}

	ps.conn = conn
	// TODO: Stop in case of error or cancel, send messages here...
	// go ps.readLoop()

	return nil
}

func (ps *SCIONOptimizedSocket) Close() error {
	return ps.conn.Close()
}

func (ps *SCIONOptimizedSocket) Conn() net.PacketConn {
	return ps.conn
}

func (ps *SCIONOptimizedSocket) SetDeadline(t time.Time) error {
	return ps.conn.SetDeadline(t)
}
func (ps *SCIONOptimizedSocket) SetReadDeadline(t time.Time) error {
	return ps.conn.SetReadDeadline(t)
}
func (ps *SCIONOptimizedSocket) SetWriteDeadline(t time.Time) error {
	return ps.conn.SetWriteDeadline(t)
}
