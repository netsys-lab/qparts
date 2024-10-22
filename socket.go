package qparts

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/scionproto/scion/pkg/snet"
	//"time"
)

type SCIONSocketPacket struct {
	Data    []byte
	NextHop netip.AddrPort
}

func (sp *SCIONSocketPacket) SetPath(path *PartsPath) {

}

type PartsSocket struct {
	sync.Mutex
	localAddr string
	underlay  UnderlaySCIONSocket
	parser    *SCIONPacketParser
}

func NewPartsSocket(local string) (*PartsSocket, error) {

	// addr := netip.MustParseAddrPort(local)
	addr, err := snet.ParseUDPAddr(local)
	if err != nil {
		return nil, err
	}
	us := NewSCIONOptimizedSocket()
	err = us.Listen(addr.Host.AddrPort())
	if err != nil {
		return nil, err
	}

	Log.Info("Listening on ", local)

	partsConn := &PartsSocket{
		localAddr: local,
		underlay:  us,
		parser:    NewSCIONPacketParser(),
	}

	return partsConn, nil
}

func (p *PartsSocket) AcceptConn() (*PartsConn, error) {

	for {

		// buf := make([]byte, 1400)
		//bts := 0
		//total := 0
		/*go func() {
			for {
				// Calculate mbit/s out of bts
				Log.Info("Throughput: ", bts*8/1000000, " Mbit/s")
				Log.Info("Total: ", total, " bytes")
				bts = 0
				time.Sleep(1 * time.Second)
			}
		}()*/
		packets, err := p.underlay.ReadMany()
		/*for {
			packets, err := p.underlay.ReadMany()
			if err != nil {
				return nil, err
			}
			bts += len(packets) * 1300
			total += len(packets) * 1300
		}*/

		Log.Info("Got packets")

		// TODO: Parallel?
		for _, packet := range packets {
			/*sbdPayload, err := p.parser.Parse(len(packet.Data), packet.Data)
			if err != nil {
				return nil, err
			}*/

			hs := NewSbdHandshake()
			hs.raw = packet.Payload
			err = hs.Decode()
			if err != nil {
				return nil, err
			}

			retHs := NewSbdHandshake()
			retHs.LocalBase = p.localAddr
			retHs.LocalConn = NewLocalAddr(p.localAddr)
			retHs.RemoteBase = hs.LocalBase
			retHs.RemoteConn = hs.LocalConn

			fmt.Printf("Sending back handshake localBAse %s, localConn %s, remoteBase %s, remoteConn %s\n", retHs.LocalBase, retHs.LocalConn, retHs.RemoteBase, retHs.RemoteConn)

			err = retHs.Encode()
			if err != nil {
				return nil, err
			}

			// Log.Info(hs)

			// Write to SCION, retry?
			// ad := addr.MustParseAddr(retHs.RemoteBase)

			destAddr, err := snet.ParseUDPAddr(retHs.RemoteConn)
			if err != nil {
				return nil, err
			}

			Log.Info("Query Paths to ", destAddr)
			paths, err := QueryPaths(destAddr.IA)
			if err != nil {
				return nil, err
			}
			Log.Info("Paths ", paths)

			/*lAddr, err := snet.ParseUDPAddr(retHs.LocalBase)
			if err != nil {
				return nil, err
			}

			udpAddr := lAddr.Host

			// baseDestAddrStr := destAddr.IA.String() + "," + retHs.RemoteBase
			// Log.Info(baseDestAddrStr)
			baseDestAddr, err := snet.ParseUDPAddr(retHs.RemoteBase)
			if err != nil {
				return nil, err
			}
			serializer, err := NewSCIONPacketSerializer(destAddr.IA, udpAddr, baseDestAddr, paths[0].Internal.Dataplane())
			if err != nil {
				return nil, err
			}

			data, err := serializer.Serialize(retHs.raw)
			if err != nil {
				return nil, err
			}

			addrp, _ := netip.ParseAddrPort(paths[0].Internal.UnderlayNextHop().String())
			packet := &SCIONSocketPacket{
				Data:    data,
				NextHop: addrp, // paths[0].Internal.UnderlayNextHop().AddrPort(),
			}

			if !packet.NextHop.IsValid() {
				Log.Info("TESTASDSAD2")
				Log.Info(baseDestAddr.Host.String())
				ad, err := netip.ParseAddrPort(baseDestAddr.Host.String())
				if err != nil {
					return nil, err
				}
				packet.NextHop = ad
				Log.Info("NEXTHOP")
				Log.Info(packet.NextHop)
				// packet.NextHop.Port = 30041
			}

			/*conn := &snet.SCIONPacketConn{
				Conn: p.underlay.Conn(),
			}

			conn.WriteTo(packet, destAddr)*/

			baseDestAddr, err := snet.ParseUDPAddr(retHs.RemoteBase)
			if err != nil {
				return nil, err
			}
			packet.Payload = retHs.raw
			packet.Addr = baseDestAddr
			packet.Path = paths[0].Internal
			Log.Info("Sending back response to ", baseDestAddr)

			_, err = p.underlay.WriteMany([]*SCIONPacket{packet})
			if err != nil {
				return nil, err
			}

			connDestAddr, err := snet.ParseUDPAddr(retHs.RemoteConn)
			if err != nil {
				return nil, err
			}

			connLocalAddr, err := snet.ParseUDPAddr(retHs.LocalConn)
			if err != nil {
				return nil, err
			}

			pc, err := NewPartsConn(connLocalAddr, connDestAddr)
			if err != nil {
				return nil, err
			}

			pc.remote = destAddr
			return pc, nil

		}
	}
}

func (p *PartsSocket) OpenConn(remote string) (*PartsConn, error) {

	// Initialize ControlPlane
	// Dial to other CP, send handshake

	Log.Info(remote)
	hs := NewSbdHandshake()
	hs.LocalBase = p.localAddr
	hs.LocalConn = NewLocalAddr(p.localAddr)

	err := hs.Encode()
	if err != nil {
		return nil, err
	}
	sAddr, err := snet.ParseUDPAddr(remote)
	if err != nil {
		return nil, err
	}
	fmt.Print(remote)

	paths, err := QueryPaths(sAddr.IA)
	if err != nil {
		return nil, err
	}
	Log.Info("Paths ", paths)
	// TODO: MTU
	buf := make([]byte, 1300)
	copy(buf, hs.raw)

	packet := &SCIONPacket{
		Payload: buf, // hs.raw,
		/// paths[0].Internal.UnderlayNextHop().AddrPort(),
		Addr: sAddr,
	}
	if len(paths) > 0 {
		packet.Path = paths[0].Internal
	}
	Log.Info("Sending handshake over Path")
	Log.Info("Path ", packet.Path)

	tries := 0

	for tries < 3 {
		p.underlay.SetReadDeadline(time.Now().Add(3 * time.Second))
		_, err = p.underlay.WriteMany([]*SCIONPacket{packet})
		if err != nil {
			return nil, err
		}

		for {
			packets, err := p.underlay.ReadMany()
			if err != nil && !strings.Contains(err.Error(), "timeout") {
				return nil, err
			}

			if err != nil && strings.Contains(err.Error(), "timeout") {
				// Move to outer loop to try again
				// TODO: Logging
				break
			}

			Log.Info("Got packets from resposne")
			// TODO: Parallel?
			for _, packet := range packets {
				/*Log.Info(packet.Data)
				sbdPayload, err := p.parser.Parse(len(packet.Data), packet.Data)
				if err != nil {
					return nil, err
				}*/
				Log.Info("TEST")
				// Log.Info(packet.Payload)
				retHs := NewSbdHandshake()
				retHs.raw = packet.Payload
				err = retHs.Decode()
				if err != nil {
					return nil, err
				}

				Log.Info("TEST2")

				fmt.Printf("Local handshake localBAse %s, localConn %s, remoteBase %s, remoteConn %s\n", hs.LocalBase, hs.LocalConn, hs.RemoteBase, hs.RemoteConn)
				fmt.Printf("Received handshake localBAse %s, localConn %s, remoteBase %s, remoteConn %s\n", retHs.LocalBase, retHs.LocalConn, retHs.RemoteBase, retHs.RemoteConn)

				connDestAddr, err := snet.ParseUDPAddr(retHs.LocalConn)
				if err != nil {
					return nil, err
				}

				connLocalAddr, err := snet.ParseUDPAddr(hs.LocalConn)
				if err != nil {
					return nil, err
				}

				pc, err := NewPartsConn(connLocalAddr, connDestAddr)
				if err != nil {
					return nil, err
				}

				// pc.remote = destAddr
				return pc, nil

			}
			tries++
		}

	}

	return nil, errors.New(PartsError[PARTS_ERROR_HANDSHAKE_TIMEOUT])
}

/*
func (p *PartsSocket) sendStuff(data []byte, nextHop netip.AddrPort, path snet.Path, src *snet.UDPAddr, dst *snet.UDPAddr) error {

	bts := make([]byte, 1200)

	pkt := &snet.Packet{
		Bytes: bts,
		PacketInfo: snet.PacketInfo{
			Source: snet.SCIONAddress{
				IA:   addr.IA(src.IA),
				Host: addr.HostIP(src.Host.AddrPort().Addr()),
			},
			Destination: snet.SCIONAddress{
				IA:   addr.IA(dst.IA),
				Host: addr.HostIP(dst.Host.AddrPort().Addr()),
			},
			Path: path.Dataplane(),
			Payload: snet.UDPPayload{
				SrcPort: src.Host.AddrPort().Port(),
				DstPort: dst.Host.AddrPort().Port(),
				Payload: data,
			},
		},
	}

	conn := &snet.SCIONPacketConn{
		Conn: p.underlay.Conn(),
	}

	err := conn.WriteTo(pkt, net.UDPAddrFromAddrPort(nextHop))
	if err != nil {
		return err
	}

	return nil
}*/

const (
	SBD_SOCK_PACKET_HS = 1
)

type SbdHandshakePacket struct {
	Flags      uint32
	RemoteBase string
	RemoteConn string
	LocalBase  string
	LocalConn  string
	raw        []byte
}

func NewSbdHandshake() *SbdHandshakePacket {
	hs := SbdHandshakePacket{
		raw: make([]byte, PACKET_SIZE),
	}
	return &hs
}

func (hs *SbdHandshakePacket) Decode() error {
	flags := binary.BigEndian.Uint32(hs.raw)
	network := bytes.NewBuffer(hs.raw[4:])
	dec := gob.NewDecoder(network)
	err := dec.Decode(hs)
	if err != nil {
		return err
	}
	hs.Flags = flags
	return nil
}

func (hs *SbdHandshakePacket) Encode() error {
	var network bytes.Buffer        // Stand-in for a network connection
	enc := gob.NewEncoder(&network) // Will write to network.
	err := enc.Encode(hs)
	if err != nil {
		return err
	}
	binary.BigEndian.PutUint32(hs.raw, hs.Flags)
	copy(hs.raw[4:], network.Bytes())
	return nil
}
