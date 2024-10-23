package qpnet

import "github.com/scionproto/scion/pkg/snet"

type CongestionEvent struct {
	Remote     *snet.UDPAddr
	EventType  string
	PathId     uint32
	Throughput uint32
	PacketLoss uint32
}
