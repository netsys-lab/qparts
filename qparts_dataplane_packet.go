package qparts

import "encoding/binary"

type QPartsDataplanePacket struct {
	Flags     uint32
	StreamId  uint64
	PartId    uint64
	FrameId   uint64
	FrameSize uint64
	Data      []byte
}

func NewQPartsDataplanePacket() *QPartsDataplanePacket {
	return &QPartsDataplanePacket{
		Data: make([]byte, 36),
	}
}

func (packet *QPartsDataplanePacket) GetHeaderLen() int {
	return 36
}

func (packet *QPartsDataplanePacket) Encode() {
	// Log.Info("Bufferlen ", len(*buf))
	// TODO: Maybe set flags here
	binary.BigEndian.PutUint32(packet.Data[0:4], packet.Flags)
	binary.BigEndian.PutUint64(packet.Data[4:12], packet.StreamId)
	binary.BigEndian.PutUint64(packet.Data[12:20], packet.PartId)
	binary.BigEndian.PutUint64(packet.Data[20:28], packet.FrameId)
	binary.BigEndian.PutUint64(packet.Data[28:36], packet.FrameSize)
}

func (packet *QPartsDataplanePacket) Decode() {

	packet.Flags = binary.BigEndian.Uint32(packet.Data[0:4])
	packet.StreamId = binary.BigEndian.Uint64(packet.Data[4:12])
	packet.PartId = binary.BigEndian.Uint64(packet.Data[12:20])
	packet.FrameId = binary.BigEndian.Uint64(packet.Data[20:28])
	packet.FrameSize = binary.BigEndian.Uint64(packet.Data[28:36])

}
