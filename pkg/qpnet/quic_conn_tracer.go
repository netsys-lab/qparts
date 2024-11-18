package qpnet

import (
	"context"
	"fmt"

	"github.com/netsys-lab/qparts/pkg/qpscion"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/logging"
	"github.com/scionproto/scion/pkg/snet"
)

type QPartsQuicTracer struct {
	Tracer       *logging.ConnectionTracer
	Context      context.Context
	Perspective  logging.Perspective
	ConnectionID quic.ConnectionID
	Path         *qpscion.QPartsPath
	Destination  *snet.UDPAddr
}

func NewQPartsQuicTracer() *QPartsQuicTracer {
	return &QPartsQuicTracer{}
}

var NewTracerFunc func(context context.Context, perspective logging.Perspective, connectionID quic.ConnectionID) *logging.ConnectionTracer

func (qt *QPartsQuicTracer) NewTracerHandler() func(context context.Context, perspective logging.Perspective, connectionID quic.ConnectionID) *logging.ConnectionTracer {
	return func(context context.Context, perspective logging.Perspective, connectionID quic.ConnectionID) *logging.ConnectionTracer {
		qt.Context = context
		qt.Perspective = perspective
		qt.ConnectionID = connectionID

		qt.Tracer = &logging.ConnectionTracer{}
		qt.Tracer.AcknowledgedPacket = func(encLevel logging.EncryptionLevel, packetNumber logging.PacketNumber) {
			fmt.Println("Acked Packet")
		}
		qt.Tracer.LostPacket = func(encLevel logging.EncryptionLevel, packetNumber logging.PacketNumber, reason logging.PacketLossReason) {
			fmt.Println("Lost Packet")
			fmt.Println(reason)
		}
		return qt.Tracer
	}
}

func (qt *QPartsQuicTracer) SetPath(path *qpscion.QPartsPath) {
	qt.Path = path
}

func (qt *QPartsQuicTracer) SetDestination(dest *snet.UDPAddr) {
	qt.Destination = dest
}

/*
var tracers := map[quic.ConnectionID]QPartsQuicTracer{}
conf := quic.Config{
	Tracer: func(context context.Context, perspective logging.Perspective, connectionID quic.ConnectionID) *logging.ConnectionTracer {
		fmt.Println("Obtain Tracer")
		t := QPartsQuicTracer{}

		tracer := &logging.ConnectionTracer{}
		tracer.AcknowledgedPacket = func(encLevel logging.EncryptionLevel, packetNumber logging.PacketNumber) {
			fmt.Println("Acked Packet")
		}
		tracer.LostPacket = func(encLevel logging.EncryptionLevel, packetNumber logging.PacketNumber, reason logging.PacketLossReason) {
			fmt.Println("Lost Packet")
			fmt.Println(reason)
		}
		t.Tracer = *tracer
		t.Context = context
		t.Perspective = perspective
		t.ConnectionID = connectionID
		tracers[connectionID] = t
		return tracer
	},
}*/
