/*
 * Packs and unpacks parts packets as follows
 * Data Packet:
 * 	- Flags: 4 bytes
 * 	- Stream ID: 8 bytes
 * 	- Sequence Number: 8 bytes
 * 	- Path ID: 4 bytes
 *  - Part ID: 8 bytes
 *  - PartSize: 4 bytes
 * 	- Data: variable length
 *
 * Ack Packet:
 * 	- Flags: 4 bytes
 * 	- Stream ID: 8 bytes
 * 	- Sequence Number: 8 bytes
 * 	- Path ID: 4 bytes
 *  - Part ID: 8 bytes
 *  - LastAckedSequenceNo: 8 bytes
 *  - NumNacks: 4 bytes
 *  - Nacks: variable length
 *
 * Metric Packet:
 * 	- Flags: 4 bytes
 * 	- Stream ID: 8 bytes
 * 	- Sequence Number: 8 bytes
 * 	- Path ID: 4 bytes
 *  - Part ID: 8 bytes
 *  - Throughput: 4 bytes
 *  - PacketLoss: 4 bytes
 *
 * Control Handshake Packet:
 * 	- Flags: 4 bytes
 * TBD:

 * Retransfer Packet: Data Packet with new SequenceNo
 *
 */

package qparts

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"net"

	"github.com/scionproto/scion/pkg/addr"
	"github.com/scionproto/scion/pkg/snet"
)

type PartsPacketPacker struct {
}

func NewPartsPacketPacker() *PartsPacketPacker {
	return &PartsPacketPacker{}
}

/*
 * Data Packet Handling:
 * 	- Flags: 4 bytes
 * 	- Stream ID: 8 bytes
 * 	- Sequence Number: 8 bytes
 * 	- Path ID: 4 bytes
 *  - Part ID: 8 bytes
 *  - PartSize: 4 bytes
 * 	- Data: variable length
 */
func (bp *PartsPacketPacker) GetDataHeaderLen() int {
	return 36
}

func (bp *PartsPacketPacker) PackData(buf *[]byte, packet *PartsDataPacket) error {
	// Log.Info("Bufferlen ", len(*buf))
	// TODO: Maybe set flags here
	binary.BigEndian.PutUint32((*buf)[0:4], packet.Flags)
	binary.BigEndian.PutUint64((*buf)[4:12], packet.StreamId)
	binary.BigEndian.PutUint64((*buf)[12:20], packet.SequenceNumber)
	binary.BigEndian.PutUint32((*buf)[20:24], packet.PathId)
	binary.BigEndian.PutUint64((*buf)[24:32], packet.PartId)
	binary.BigEndian.PutUint32((*buf)[32:36], packet.PartSize)
	return nil
}

func (bp *PartsPacketPacker) UnpackData(buf *[]byte) (*PartsDataPacket, error) {
	p := PartsDataPacket{}
	p.Flags = binary.BigEndian.Uint32((*buf)[0:4])
	p.StreamId = binary.BigEndian.Uint64((*buf)[4:12])
	p.SequenceNumber = binary.BigEndian.Uint64((*buf)[12:20])
	p.PathId = binary.BigEndian.Uint32((*buf)[20:24])
	p.PartId = binary.BigEndian.Uint64((*buf)[24:32])
	p.PartSize = binary.BigEndian.Uint32((*buf)[32:36])

	*buf = (*buf)[36:]
	p.Data = *buf
	return &p, nil
}

/*
 * Handshake Packet Handling:
 */
type PartsHandshakePacket struct {
	Flags    int32
	StreamId uint64
	raw      []byte
}

func NewPartsHandshake() *PartsHandshakePacket {
	hs := PartsHandshakePacket{
		raw: make([]byte, PACKET_SIZE),
	}
	hs.prepareFlags()
	return &hs
}

func (hs *PartsHandshakePacket) prepareFlags() {
	hs.Flags = NewPartsFlags()
}

func (hs *PartsHandshakePacket) Decode() error {
	flags := int32(binary.BigEndian.Uint32(hs.raw))
	network := bytes.NewBuffer(hs.raw[4:])
	dec := gob.NewDecoder(network)
	err := dec.Decode(hs)
	if err != nil {
		return err
	}
	hs.Flags = flags
	// TODO: Maybe we need this one, too
	// hs.raw = network.Bytes()
	return nil
}

func (hs *PartsHandshakePacket) Encode() error {
	var network bytes.Buffer        // Stand-in for a network connection
	enc := gob.NewEncoder(&network) // Will write to network.
	err := enc.Encode(hs)
	if err != nil {
		return err
	}
	binary.BigEndian.PutUint32(hs.raw, uint32(hs.Flags))
	copy(hs.raw[4:], network.Bytes())
	return nil
}

/* SCION */

type SCIONPacketSerializer struct {
	baseBytes        snet.Bytes
	headerBytes      int
	basePayloadBytes int

	listenAddr *net.UDPAddr
	remoteAddr *snet.UDPAddr
}

const SCION_PROTOCOL_NUMBER_SCION_UDP = 17

func NewSCIONPacketSerializer(localIA addr.IA, listenAddr *net.UDPAddr, remoteAddr *snet.UDPAddr, path snet.DataplanePath) (*SCIONPacketSerializer, error) {

	// TODO: Fix proper parsing here
	scionDestinationAddress := snet.SCIONAddress{
		IA:   remoteAddr.IA,
		Host: addr.MustParseHost(remoteAddr.Host.IP.String()),
	}

	scionListenAddress := snet.SCIONAddress{
		IA:   localIA,
		Host: addr.MustParseHost(listenAddr.IP.String()),
	}

	var bytes snet.Bytes
	bytes.Prepare()

	preparedPacket := &snet.Packet{
		Bytes: bytes,
		PacketInfo: snet.PacketInfo{
			Destination: scionDestinationAddress,
			Source:      scionListenAddress,
			Path:        path,
			// This is a hack.
			Payload: snet.UDPPayload{
				Payload: make([]byte, 0),
				SrcPort: 0,
				DstPort: 0,
			},
		},
	}

	err := preparedPacket.Serialize()
	if err != nil {
		return nil, err
	}

	headerBytes := len(preparedPacket.Bytes) - 8
	// We use the Packet to calculate the correct sum and subtract our dummy payload.
	basePayloadBytes := int(binary.BigEndian.Uint16(preparedPacket.Bytes[6:8]) - 8)

	pS := SCIONPacketSerializer{
		listenAddr:       listenAddr,
		remoteAddr:       remoteAddr,
		baseBytes:        preparedPacket.Bytes,
		headerBytes:      headerBytes,
		basePayloadBytes: basePayloadBytes,
	}

	return &pS, nil
}

func (pS *SCIONPacketSerializer) Serialize(b []byte) ([]byte, error) {

	l4PayloadSize := 8 + len(b)

	// Network Byte Order is Big Endian
	binary.BigEndian.PutUint16(pS.baseBytes[6:8], uint16(pS.basePayloadBytes+l4PayloadSize))
	// Log.Info("Setting remoteAddr port: ", pS.remoteAddr.Host.Port)
	binary.BigEndian.PutUint16(pS.baseBytes[pS.headerBytes+0:pS.headerBytes+2], uint16(pS.listenAddr.Port))
	binary.BigEndian.PutUint16(pS.baseBytes[pS.headerBytes+2:pS.headerBytes+4], uint16(pS.remoteAddr.Host.Port))
	binary.BigEndian.PutUint16(pS.baseBytes[pS.headerBytes+4:pS.headerBytes+6], uint16(l4PayloadSize))
	binary.BigEndian.PutUint16(pS.baseBytes[pS.headerBytes+6:pS.headerBytes+8], uint16(0))

	copy(pS.baseBytes[pS.headerBytes+8:pS.headerBytes+l4PayloadSize], b)

	dataLength := pS.headerBytes + l4PayloadSize
	return pS.baseBytes[0:dataLength], nil
}

func (pS *SCIONPacketSerializer) GetHeaderLen() int {
	// ps.HeaderBytes contains the header length without the UDP header.
	// An UDP header is 8 bytes long.
	return pS.headerBytes + 8
}

type SCIONPacketParser struct {
	ReadBuffer []byte
}

func NewSCIONPacketParser() *SCIONPacketParser {

	readBuffer := make([]byte, 9000) //

	SCIONPacketParser := SCIONPacketParser{
		ReadBuffer: readBuffer,
	}

	return &SCIONPacketParser
}

func (pP *SCIONPacketParser) Parse(n int, readBytes []byte) ([]byte, error) {
	// Payload is L4 UDP, we need to unpack this too. This has a fixed length of 8 bytes.
	nextHdr := uint16(readBytes[4])

	if nextHdr != SCION_PROTOCOL_NUMBER_SCION_UDP {
		return nil, errors.New("Unknown Packet")
	}

	udpPayloadLen := int(binary.BigEndian.Uint16(readBytes[6:8]))

	payloadLen := udpPayloadLen - 8
	startPos := n - payloadLen

	// copy(readBytes, readBytes[startPos:n])

	return readBytes[startPos:n], nil
}
