package qparts

import (
	"context"
	"encoding/binary"

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

func NewPartsFlags() int32 {
	var flags int32 = 0
	flags = AddVersionFlag(PARTS_VERSION, MASK_FLAGS_VERSION)
	return flags
}

func AddMsgFlag(val, flag int32) int32 {
	return val | (flag & MASK_FLAGS_MSG)
}

func AddVersionFlag(val, flag int32) int32 {
	return val | (flag & MASK_FLAGS_VERSION)
}

const (
	DP_STATE_SENDING    = 100
	DP_STATE_RETRANSFER = 101
	DP_STATE_PENDING    = 200
	DP_STATE_RECEIVING  = 300
)

const (
	PACKET_SIZE         = 600
	PACKET_PAYLOAD_SIZE = 900 // TODO: Calculate this live...
)

type PartRequestPacket struct {
	PartId                       int64
	LastSequenceNumber           int64
	NumPacketsPerTx              int
	PacketTxIndex                int
	MissingSequenceNumbers       []int64
	MissingSequenceNumberOffsets []int64
	TransactionId                int64
	RequestSequenceId            int
	NumPackets                   int64
	PartFinished                 bool
}

type PartsDataPacket struct {
	Flags          uint32
	StreamId       uint64
	PartId         uint64
	PartSize       uint32
	PathId         uint32
	SequenceNumber uint64
	Data           []byte
}

type PartsDataplane struct {
	us          UnderlaySCIONSocket
	streams     map[uint64]*PartsStream
	cp          *SbdControlPlane
	parser      *SCIONPacketParser
	packer      *PartsPacketPacker
	local       *snet.UDPAddr
	remote      *snet.UDPAddr
	serializer  map[string]*SCIONPacketSerializer
	completions map[uint64]*PartCompletion
	partIds     map[uint64]uint64
}

func NewDataplane(local, remote *snet.UDPAddr, streams map[uint64]*PartsStream) *PartsDataplane {
	// log.Fatal("NEW DATAPLANE")
	dp := &PartsDataplane{
		streams:     streams,
		parser:      NewSCIONPacketParser(),
		packer:      NewPartsPacketPacker(),
		local:       local,
		remote:      remote,
		serializer:  make(map[string]*SCIONPacketSerializer),
		completions: make(map[uint64]*PartCompletion),
		partIds:     make(map[uint64]uint64),

		// TODO: Ensure partIds filled with proper value for each path
	}

	return dp
}

func (sbd *PartsDataplane) SetControlPlane(cp *SbdControlPlane) {
	sbd.cp = cp
}

func (sbd *PartsDataplane) RemoveStream(id uint64) {
	delete(sbd.streams, id)
}

func (sbd *PartsDataplane) Run() error {
	sbd.us = NewSCIONOptimizedSocket()
	err := sbd.us.Listen(sbd.local.Host.AddrPort())
	if err != nil {
		return err
	}
	go sbd.readLoop()
	return nil
}

func (sbd *PartsDataplane) processPacket(p *SCIONPacket) {
	/*sbdPayload, err := sbd.parser.Parse(len(p.Data), p.Data)
	if err != nil {
		sbd.cp.ErrChan <- err
		return
	}*/

	flags := binary.BigEndian.Uint32(p.Payload[0:4])

	switch flags {

	case PARTS_MSG_DATA:
		packet, err := sbd.packer.UnpackData(&p.Payload)
		if err != nil {
			sbd.cp.ErrChan <- err
			return
		}

		Log.Info("Received from stream ", packet.StreamId)
		Log.Info("Part ", packet.PartId)
		Log.Info("SequenceNo ", packet.SequenceNumber)

		/*
		 *
		 */
		// Log.Info("Received from path ", packet.PathId)
		//Log.Info(packet)
		// TODO: THis is not working
		//sbd.cp.TempPacketChan <- packet.PathId
		sbd.streams[packet.StreamId].ReadBuffer.Enqueue(packet.Data)

	case PARTS_MSG_ACK:
		metricPacket := SbdMetric{
			Throughput: binary.BigEndian.Uint32(p.Payload[8:12]),
			PacketLoss: binary.BigEndian.Uint32(p.Payload[12:16]),
			PathId:     binary.BigEndian.Uint32(p.Payload[4:8]),
		}
		sbd.cp.MetricPacketChan <- metricPacket
	case PARTS_MSG_HS:
		Log.Info("HS PACKET")
		sbd.cp.ControlHandshakePacketChan <- p.Payload
	case PARTS_MSG_RT:
		break
	case PARTS_MSG_ACK_CT:
		break
	default:

	}
	// TODO: Do stuff here with the packet
	// TODO: Fix size of read buffer
	// sbd.streams[packet.StreamId].ReadBuffer.Add(packet.Data)
}

func (sbd *PartsDataplane) readLoop() {
	Log.Info("READLOOP")
	for {
		// Read from underlay socket
		// Store in stream buffers

		// Check for cancel messages
		packets, err := sbd.us.ReadMany()
		if err != nil {
			sbd.cp.ErrChan <- err
			continue
		}

		// TODO: Error handling
		for _, p := range packets {
			// Log.Info("Processing packet ")
			sbd.processPacket(p)
		}

	}
}

func (sbd *PartsDataplane) GetNextPartId(streamId uint64) uint64 {
	_, ok := sbd.partIds[streamId]
	if !ok {
		sbd.partIds[streamId] = 0
	}

	sbd.partIds[streamId]++
	return sbd.partIds[streamId]
}

func (sbd *PartsDataplane) WriteForStream(schedulingDecision *SchedulingDecision, id uint64) (int, error) {

	numSents := 0
	for _, assignment := range schedulingDecision.Assignments {
		sps := make([]*SCIONPacket, 0)
		/*
		 * Add part completions here, think about parallelism
		 * but start with sequential writing here
		 */

		// Calculate how many packets are needed considering len(assignment.Data) and assignment.Path.MTU

		numPackets := CeilForceInt(len(assignment.Data), assignment.Path.PayloadSize)
		// TODO: Get Sequence Number from Stream
		partId := sbd.GetNextPartId(id)
		partCompletion := &PartCompletion{
			PartId:     partId,
			StreamId:   id,
			SequenceNo: 0,          // TODO: Implement Sequence Number Spaces
			Size:       numPackets, // uint64(len(assignment.Data)), // TODO: Round properly
			Data:       assignment.Data,
			PathId:     assignment.Path.Id,
		}

		partsDataPacket := &PartsDataPacket{
			Flags:    PARTS_MSG_DATA, // uint32(NewPartsFlags()),
			StreamId: id,
			PartId:   partId,
			PartSize: uint32(len(assignment.Data)),
			PathId:   assignment.Path.Id,
		}

		for i := 0; i < numPackets; i++ {
			// Get data from assignment.Data
			// Create a new PartsDataPacket
			// Pack the data
			// Send the packet
			target := (i + 1) * assignment.Path.PayloadSize
			data := assignment.Data[i*assignment.Path.PayloadSize : target]
			buf := make([]byte, 1200 /*len(assignment.Data)+sbd.packer.GetDataHeaderLen()*/)

			if target > len(assignment.Data) {
				target = len(assignment.Data)
			}

			partsDataPacket.SequenceNumber = partCompletion.NextSequenceNo()
			copy(buf[sbd.packer.GetDataHeaderLen():], data)

			err := sbd.packer.PackData(&buf, partsDataPacket)
			if err != nil {
				return 0, err
			}

			sp := &SCIONPacket{
				Payload: buf,
				Path:    assignment.Path.Internal,
				Addr:    assignment.Remote,
			}

			sps = append(sps, sp)
		}

		sbd.completions[partId] = partCompletion

		res, err := sbd.us.WriteMany(sps)
		if err != nil {
			return res, err
		}
		numSents += res

	}

	return numSents, nil
}

func (sbd *PartsDataplane) WriteRaw(data []byte, remote *snet.UDPAddr) (int, error) {

	hc := host()

	// TODO: Cache
	paths, err := hc.queryPaths(context.Background(), remote.IA)
	if err != nil {
		return 0, err
	}

	sp := &SCIONPacket{
		Payload: data,
		Path:    paths[0],
		Addr:    remote,
	}

	res, err := sbd.us.WriteMany([]*SCIONPacket{sp})
	return res, err
}
