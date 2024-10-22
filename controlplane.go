package qparts

import (
	"encoding/binary"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/scionproto/scion/pkg/snet"
)

type SbdControlPlane struct {
	ErrChan                    chan error
	ControlHandshakePacketChan chan []byte
	TempPacketChan             chan uint32
	MetricPacketChan           chan SbdMetric
	streams                    map[uint64]*PartsStream
	local                      *snet.UDPAddr
	remote                     *snet.UDPAddr
	dp                         *PartsDataplane
	Scheduler                  *Scheduler
	Conn                       *PartsConn
	ControlConn                *SingleStreamQUICConn
}

type SbdMetric struct {
	Throughput uint32
	PacketLoss uint32
	PathId     uint32
}

func NewControlPlane(local, remote *snet.UDPAddr, streams map[uint64]*PartsStream, dataplane *PartsDataplane) *SbdControlPlane {
	return &SbdControlPlane{
		ErrChan:                    make(chan error),
		ControlHandshakePacketChan: make(chan []byte),
		TempPacketChan:             make(chan uint32),
		MetricPacketChan:           make(chan SbdMetric),
		remote:                     remote,
		local:                      local,
		streams:                    streams,
		dp:                         dataplane,
	}
}

func (cp *SbdControlPlane) RemoveStream(id uint64) {
	delete(cp.streams, id)
}

func logMetric(metric SbdMetric, local, remote string) {
	Log.Info("Received new metric data over path ", metric.PathId)
	Log.Info("Throughput: ", metric.Throughput)
	Log.Info("PacketLoss: ", metric.PacketLoss)

	f, err := os.OpenFile("/opt/sbd.log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}

	defer f.Close()
	bottleneck := os.Getenv("BOTTLENECK")
	strategy := os.Getenv("SCHEDULER_STRATEGY")
	numPathsEnv := os.Getenv("NUM_PATHS")
	text := fmt.Sprintf("Received new metric data from %s to %s over path %d;Throughput: %d;PacketLoss: %d;Bottleneck: %s;Strategy: %s; NumPaths: %s\n", local, remote, metric.PathId, metric.Throughput, metric.PacketLoss, bottleneck, strategy, numPathsEnv)

	if _, err = f.WriteString(text); err != nil {
		panic(err)
	}
}

func (cp *SbdControlPlane) RunMetricLoop() error {
	// paths, err := QueryPaths(cp.remote.IA)
	//if err != nil {
	//		return err/
	//}

	bottleneck := os.Getenv("BOTTLENECK")
	Log.Info(bottleneck)

	numPathsEnv := os.Getenv("NUM_PATHS")
	numPaths, _ := strconv.Atoi(numPathsEnv)

	count := 0
	for {
		metric := <-cp.MetricPacketChan
		Log.Info("Received new metric data over path ", metric.PathId)
		Log.Info("Throughput: ", metric.Throughput)
		Log.Info("PacketLoss: ", metric.PacketLoss)

		logMetric(metric, cp.local.String(), cp.remote.String())

		Log.Info(bottleneck)
		// Check if global bottleneck interface is on the paths

		if metric.PacketLoss > 0 {
			ce := &CongestionEvent{
				Remote:     cp.remote,
				PacketLoss: metric.PacketLoss,
				Throughput: metric.Throughput,
				PathId:     metric.PathId,
				EventType:  "Congestion",
			}

			err := cp.Scheduler.OnCongestionEvent(ce)
			if err != nil {
				log.Fatal(err)
			}
		}

		Log.Info("Comparing count and numPaths ", count, " : ", (7 * numPaths))
		if count >= (7 * numPaths) {
			os.Exit(0)
		}

		count++
	}
}

func Reverse[T any](original []T) (reversed []T) {
	reversed = make([]T, len(original))
	copy(reversed, original)

	for i := len(reversed)/2 - 1; i >= 0; i-- {
		tmp := len(reversed) - 1 - i
		reversed[i], reversed[tmp] = reversed[tmp], reversed[i]
	}

	return
}

func (cp *SbdControlPlane) RunCCLoop() error {
	paths, err := QueryPaths(cp.remote.IA)
	if err != nil {
		return err
	}

	for i, p := range paths {
		reverseInterfaces := Reverse(p.Interfaces)
		paths[i].Sorter = strings.Join(reverseInterfaces, ",")
	}

	SortPartsPaths(paths)

	bottleneck := os.Getenv("BOTTLENECK")
	numPathsEnv := os.Getenv("NUM_PATHS")
	numPaths, _ := strconv.Atoi(numPathsEnv)

	pcount := 0
	for {
		pathId := <-cp.TempPacketChan
		Log.Info("Received new temp data over path ", pathId)

		// Random number between 0 and 20ms
		time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)

		bottleNeckFound := false
		// Check if global bottleneck interface is on the paths
		throughput := uint32(0)
		Log.Info(bottleneck)
		Log.Info("Interfaces ", paths[pathId].Interfaces)
		for _, intf := range paths[pathId].Interfaces {
			Log.Info(intf)

			if intf == bottleneck {
				Log.Info("Bottleneck found on path ", paths[pathId].Id)
				bottleNeckFound = true
				throughput = 1
				break
			}

			parts := strings.Split(intf, "-")
			// Core ASes
			if len(parts[1]) == 3 && throughput == 0 {
				throughput = 15
			} else {
				throughput = 10
			}

		}

		// Send back MSG_FLAG_CC or something and packet loss and throughput according to the path
		// This can be calculated from the links including the bottleneck link with 1Mbit/s
		packetLoss := uint32(rand.Intn(100))
		if !bottleNeckFound {
			packetLoss = 0
		}

		// Create an empty packet
		retData := make([]byte, 1024)
		binary.BigEndian.PutUint32(retData[0:4], PARTS_MSG_ACK)
		binary.BigEndian.PutUint32(retData[4:8], pathId)
		binary.BigEndian.PutUint32(retData[8:12], throughput)
		binary.BigEndian.PutUint32(retData[12:16], packetLoss)

		addr := cp.remote.Copy()
		addr.Path = paths[pathId].Internal.Dataplane()
		Log.Info("Sending back over path: ", paths[pathId].Sorter)

		// Send the packet back
		count, err := cp.dp.WriteRaw(retData, addr)
		if err != nil {
			return err
		}
		Log.Info("Sent back temp data")
		Log.Info("Count ", count)

		Log.Info("Comparing count and numPaths ", count, " : ", (7 * pcount))

		if pcount >= (7 * numPaths) {
			os.Exit(0)
		}

		pcount++
	}
}

func (cp *SbdControlPlane) AcceptStream() (*PartsStream, error) {

	// Add remote to state
	err := State.AddRemote(cp.remote)
	if err != nil {
		return nil, err
	}

	// Await handshake packet from remote
	for {

		packetData := <-cp.ControlHandshakePacketChan
		Log.Info("Received new stream handshake")
		hs := NewPartsHandshake()
		hs.raw = packetData

		err := hs.Decode()
		if err != nil {
			return nil, err
		}
		Log.Info("Parsed new stream handshake")

		stream := NewPartsStream(hs.StreamId, cp.Scheduler)
		stream.Conn = cp.Conn
		retHs := NewPartsHandshake()
		retHs.StreamId = newStreamId()
		retHs.Flags = PARTS_MSG_HS

		err = retHs.Encode()
		if err != nil {
			return nil, err
		}
		Log.Info("Remote CP: ", cp.remote.String())
		count, err := cp.dp.WriteRaw(retHs.raw, cp.remote)
		Log.Info("Sent new stream handshake back")
		Log.Info("Count ", count)
		if err != nil {
			return nil, err
		}

		// Write data back
		return stream, nil
		// hs.StreamId

	}
}

func (cp *SbdControlPlane) OpenStream() (*PartsStream, error) {

	// Send handshake packet to remote
	err := State.AddRemote(cp.remote)
	if err != nil {
		return nil, err
	}

	hs := NewPartsHandshake()
	hs.StreamId = newStreamId()
	hs.Flags = PARTS_MSG_HS

	err = hs.Encode()
	if err != nil {
		return nil, err
	}
	Log.Info("Remote CP: ", cp.remote.String())
	time.Sleep(1 * time.Second)
	count, err := cp.dp.WriteRaw(hs.raw, cp.remote)
	Log.Info("Sent new stream handshake")
	Log.Info("Count ", count)
	if err != nil {
		return nil, err
	}

	packetData := <-cp.ControlHandshakePacketChan
	Log.Info("Received new stream handshake")
	retHs := NewPartsHandshake()
	retHs.raw = packetData

	err = retHs.Decode()
	if err != nil {
		return nil, err
	}
	Log.Info("Stream Id", retHs.StreamId)
	Log.Info("Parsed new stream handshake")

	stream := NewPartsStream(hs.StreamId, cp.Scheduler)
	stream.Conn = cp.Conn
	return stream, nil
}

func newStreamId() uint64 {
	// Generate Random number
	return rand.Uint64()
}
