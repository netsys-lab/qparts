package qpnet

import (
	"github.com/netsys-lab/qparts/pkg/qpscion"
	"github.com/scionproto/scion/pkg/snet"
)

type PathCongestion struct {
	PathId         uint32
	Path           *qpscion.QPartsPath
	ThroughputDrop float64
	PacketLoss     float64
}

type CongestionEvent struct {
	Remote          *snet.UDPAddr
	Reason          string
	CongestionPaths []PathCongestion
}
