package qparts

import "encoding/binary"

type QPartsDataplanePacket struct {
	Flags        uint32
	StreamId     uint64
	SequenceId   uint64
	SequenceSize uint64
	PartId       uint64
	PartSize     uint64
	NumParts     uint64
	Data         []byte
}

func NewQPartsDataplanePacket() *QPartsDataplanePacket {
	return &QPartsDataplanePacket{
		Data: make([]byte, 52),
	}
}

func (packet *QPartsDataplanePacket) GetHeaderLen() int {
	return 52
}

func (packet *QPartsDataplanePacket) Encode() {
	/*
		StreamId     uint64
		SequenceId   uint64
		SequenceSize uint64
		PartId       uint64
		PartSize     uint64
	*/
	binary.BigEndian.PutUint32(packet.Data[0:4], packet.Flags)
	binary.BigEndian.PutUint64(packet.Data[4:12], packet.StreamId)
	binary.BigEndian.PutUint64(packet.Data[12:20], packet.SequenceId)
	binary.BigEndian.PutUint64(packet.Data[20:28], packet.SequenceSize)
	binary.BigEndian.PutUint64(packet.Data[28:36], packet.PartId)
	binary.BigEndian.PutUint64(packet.Data[36:44], packet.PartSize)
	binary.BigEndian.PutUint64(packet.Data[44:52], packet.NumParts)
}

func (packet *QPartsDataplanePacket) Decode() {

	packet.Flags = binary.BigEndian.Uint32(packet.Data[0:4])
	packet.StreamId = binary.BigEndian.Uint64(packet.Data[4:12])
	packet.SequenceId = binary.BigEndian.Uint64(packet.Data[12:20])
	packet.SequenceSize = binary.BigEndian.Uint64(packet.Data[20:28])
	packet.PartId = binary.BigEndian.Uint64(packet.Data[28:36])
	packet.PartSize = binary.BigEndian.Uint64(packet.Data[36:44])
	packet.NumParts = binary.BigEndian.Uint64(packet.Data[44:52])
}

type QPartsNewStreamPacket struct {
	Flags          uint32
	StreamId       uint64
	StreamProperty uint32
	Data           []byte
}

func NewQPartsNewStreamPacket() *QPartsNewStreamPacket {
	return &QPartsNewStreamPacket{
		Data: make([]byte, 16),
	}
}

func (packet *QPartsNewStreamPacket) GetHeaderLen() int {
	return 16
}

func (packet *QPartsNewStreamPacket) Encode() {
	// Log.Info("Bufferlen ", len(*buf))
	// TODO: Maybe set flags here
	binary.BigEndian.PutUint32(packet.Data[0:4], packet.Flags)
	binary.BigEndian.PutUint64(packet.Data[4:12], packet.StreamId)
	binary.BigEndian.PutUint32(packet.Data[12:16], packet.StreamProperty)
}

func (packet *QPartsNewStreamPacket) Decode() {

	packet.Flags = binary.BigEndian.Uint32(packet.Data[0:4])
	packet.StreamId = binary.BigEndian.Uint64(packet.Data[4:12])
	packet.StreamProperty = binary.BigEndian.Uint32(packet.Data[12:16])
}
