package qpnet

import (
	"context"

	"github.com/netsys-lab/qparts/pkg/qplogging"
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
		qt.Tracer.UpdatedCongestionState = func(state logging.CongestionState) {
			if state == logging.CongestionStateRecovery {
				if qt.Path != nil {
					qplogging.Log.Info("Received recovery state on connection ", connectionID)
					// qplogging.Log.Info("Path: ", qt.Path)
				}
			}
		}
		qt.Tracer.LostPacket = func(encLevel logging.EncryptionLevel, packetNumber logging.PacketNumber, reason logging.PacketLossReason) {
			if qt.Path != nil {
				qplogging.Log.Info("Lost Packet with reason ", reason, " on conn ", connectionID)
				// qplogging.Log.Info("Path: ", qt.Path)
			}
		}

		qt.Tracer.UpdatedMetrics = func(rttStats *logging.RTTStats, cwnd, bytesInFlight logging.ByteCount, packetsInFlight int) {
			//qplogging.Log.Info("Updated Metrics: ")
			// qplogging.Log.Info("RTT Stats: ", rttStats)
			//qplogging.Log.Info("CWND: ", cwnd)
			//qplogging.Log.Info("Bytes in Flight: ", bytesInFlight)
			//qplogging.Log.Info("Packets in Flight: ", packetsInFlight)
			//if qt.Path != nil {
			//	qplogging.Log.Info("Path: ", qt.Path)
			//}
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
		qplogging.Log.Debug("Obtain Tracer")
		t := QPartsQuicTracer{}

		tracer := &logging.ConnectionTracer{}
		tracer.AcknowledgedPacket = func(encLevel logging.EncryptionLevel, packetNumber logging.PacketNumber) {
			qplogging.Log.Debug("Acked Packet")
		}
		tracer.LostPacket = func(encLevel logging.EncryptionLevel, packetNumber logging.PacketNumber, reason logging.PacketLossReason) {
			qplogging.Log.Debug("Lost Packet")
			qplogging.Log.Debug(reason)
		}
		t.Tracer = *tracer
		t.Context = context
		t.Perspective = perspective
		t.ConnectionID = connectionID
		tracers[connectionID] = t
		return tracer
	},
}*/
