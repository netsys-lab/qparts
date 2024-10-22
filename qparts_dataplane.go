package qparts

import (
	"fmt"

	"github.com/scionproto/scion/pkg/snet"
)

const (
	PARTS_MSG_DATA   = 1 // Data packet
	PARTS_MSG_HS     = 2 // Handshake packet
	PARTS_MSG_ACK    = 3 // Data Acknowledgement packet
	PARTS_MSG_RT     = 4 // Retransfer packet
	PARTS_MSG_ACK_CT = 5 // Control Plane Acknowledgement packet
)

const (
	MASK_FLAGS_MSG     = 0b11111111
	MASK_FLAGS_VERSION = 0b11111111 << 16
)

const (
	PARTS_VERSION = 1 << 24
)

type QPartsDataplane struct {
	Streams map[uint64]*QPartsDataplaneStream
}

type QPartsDataplaneStream struct {
	ssqc *SingleStreamQUICConn
	path snet.DataplanePath
}

func NewQPartsDataplane() *QPartsDataplane {
	return &QPartsDataplane{
		Streams: make(map[uint64]*QPartsDataplaneStream),
	}
}

func (dp *QPartsDataplane) GetStream(id uint64) *QPartsDataplaneStream {
	return dp.Streams[id]
}

func (dp *QPartsDataplane) AddDialStream(id uint64, ssqc *SingleStreamQUICConn, path snet.DataplanePath) error {
	dp.Streams[id] = &QPartsDataplaneStream{
		ssqc: ssqc,
		path: path,
	}
	fmt.Println("Added dial stream")

	return nil
}

func (dp *QPartsDataplane) AddListenStream(id uint64, ssqc *SingleStreamQUICConn) error {
	dp.Streams[id] = &QPartsDataplaneStream{
		ssqc: ssqc,
	}
	fmt.Println("Added listen stream")

	return nil
}

func (dp *QPartsDataplane) WriteForStream(schedulingDecision *SchedulingDecision, id uint64) (int, error) {
	fmt.Println("Writing to stream ", id)
	return 0, nil
}
