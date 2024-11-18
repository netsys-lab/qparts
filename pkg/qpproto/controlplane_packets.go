package qpproto

import "encoding/binary"

func NewQPartsHandshakePacket() *QPartsHandshakePacket {
	return &QPartsHandshakePacket{
		Data: make([]byte, 26),
	}
}

type QPartsHandshakePacket struct {
	Flags          uint32
	ConnId         uint64
	Version        uint64
	StartPortRange uint16
	EndPortRange   uint16
	NumStreams     uint16
	Data           []byte
}

/*
 */
func (bp *QPartsHandshakePacket) GetHeaderLen() int {
	return 26
}

func (packet *QPartsHandshakePacket) Encode() {
	// Log.Info("Bufferlen ", len(*buf))
	// TODO: Maybe set flags here
	binary.BigEndian.PutUint32(packet.Data[0:4], packet.Flags)
	binary.BigEndian.PutUint64(packet.Data[4:12], packet.ConnId)
	binary.BigEndian.PutUint64(packet.Data[12:20], packet.Version)
	binary.BigEndian.PutUint16(packet.Data[20:22], packet.StartPortRange)
	binary.BigEndian.PutUint16(packet.Data[22:24], packet.EndPortRange)
	binary.BigEndian.PutUint16(packet.Data[24:26], packet.NumStreams)
}

func (packet *QPartsHandshakePacket) Decode() {

	packet.Flags = binary.BigEndian.Uint32(packet.Data[0:4])
	packet.ConnId = binary.BigEndian.Uint64(packet.Data[4:12])
	packet.Version = binary.BigEndian.Uint64(packet.Data[12:20])
	packet.StartPortRange = binary.BigEndian.Uint16(packet.Data[20:22])
	packet.EndPortRange = binary.BigEndian.Uint16(packet.Data[22:24])
	packet.NumStreams = binary.BigEndian.Uint16(packet.Data[24:26])

}
