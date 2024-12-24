package qparts

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/netsys-lab/qparts/pkg/qplogging"
	"github.com/netsys-lab/qparts/pkg/qpmetrics"
	"github.com/netsys-lab/qparts/pkg/qpnet"
	"github.com/netsys-lab/qparts/pkg/qpproto"
	"github.com/netsys-lab/qparts/pkg/qpscion"
)

const (
	PARTS_MSG_DATA            = 1 // Data packet
	PARTS_MSG_HS              = 2 // Handshake packet
	PARTS_MSG_ACK             = 3 // Data Acknowledgement packet
	PARTS_MSG_RT              = 4 // Retransfer packet
	PARTS_MSG_ACK_CT          = 5 // Control Plane Acknowledgement packet
	PARTS_MSG_STREAM_HS       = 6 // Handshake packet for QPartsStream
	PARTS_MSG_STREAM_PROPERTY = 7 // Data packet for QPartsStream
)

const (
	QPARTS_DP_STREAM_LOW_LATENCY     = 1
	QPARTS_DP_STREAM_HIGH_THROUGHPUT = 2
)

const (
	MASK_FLAGS_MSG     = 0b11111111
	MASK_FLAGS_VERSION = 0b11111111 << 16
)

const (
	PARTS_VERSION = 1 << 24
)

type QPartsDataplane struct {
	Streams         map[uint64]*QPartsDataplaneStream
	QPartsStreams   map[uint64]*PartsStream
	scheduler       *Scheduler
	completionStore *CompletionStore
	partsSize       int
}

type QPartsDataplaneStream struct {
	sync.Mutex
	ssqc *qpnet.SingleStreamQUICConn
	// path      snet.DataplanePath
	PartsPath       *qpscion.QPartsPath
	Queue           *qpnet.Queue[DataAssignment]
	StreamType      uint32
	OptimalPartSize int
}

func NewQPartsDataplane(scheduler *Scheduler, partsStreams map[uint64]*PartsStream) *QPartsDataplane {
	return &QPartsDataplane{
		Streams:         make(map[uint64]*QPartsDataplaneStream),
		scheduler:       scheduler,
		QPartsStreams:   partsStreams,
		completionStore: NewCompletionStore(),
		partsSize:       1000024,
	}
}

func (n *QPartsDataplane) DetectCongestion() {
	allPaths := make([]qpmetrics.PathMetrics, 0)
	for _, rem := range qpmetrics.State.Remotes {
		for _, path := range rem {

			if path.Path.Id == 0 || path.Throughput == 0 {
				continue
			}

			allPaths = append(allPaths, *path)
		}
	}

	simPaths := FindSimilarPaths(allPaths, 0.1)
	fmt.Println("Similar Paths:")
	for _, pair := range simPaths {
		fmt.Printf("Pair: PathId %s and PathId %s\n", pair[0].Path.Sorter, pair[1].Path.Sorter)
	}
}

func (dp *QPartsDataplane) StartMeasurements() {
	go dp.runDetectionTicker()
}

func (dp *QPartsDataplane) runDetectionTicker() {
	ticker := time.NewTicker(500 * time.Millisecond)

	for range ticker.C {
		qplogging.Log.Debug("Running stats check")
		dp.DetectCongestion()

		for _, rem := range qpmetrics.State.Remotes {
			for _, path := range rem {
				// tr := float64(path.Throughput*8*2) / 1024 / 1024
				// qplogging.Log.Debugf("Path %d: RTT: %d, Packetloss: %d, Throughput: %f Mbit/s, CwndDecreaseEvents: %d, PacketLossEvents: %d", path.PathId, path.RTT, path.Packetloss, tr, path.CwndDecreaseEvents, path.PacketLossEvents)
				//qplogging.Log.Debugf("Path %s: RTT: %d, Packetloss: %d, Throughput: %d, CwndDecreaseEvents: %d, PacketLossEvents: %d", path.Sorter, path.RTT, path.Packetloss, path.Throughput, path.CwndDecreaseEvents, path.PacketLossEvents)
				path.Throughput = 0
				path.Packetloss = 0
			}
		}

	}
}

func (dp *QPartsDataplane) GetStream(id uint64) *QPartsDataplaneStream {
	return dp.Streams[id]
}

func (dp *QPartsDataplane) AddDialStream(id uint64, ssqc *qpnet.SingleStreamQUICConn, path *qpscion.QPartsPath) error {
	dp.Streams[id] = &QPartsDataplaneStream{
		ssqc:      ssqc,
		PartsPath: path,
		Queue:     qpnet.NewQueue[DataAssignment](),
	}
	qplogging.Log.Debug("Added dial stream")

	return nil
}

func (dp *QPartsDataplane) ScheduleWrite(data []byte, stream *PartsStream) SchedulingDecision {
	// TODO: State
	//Log.Info("Scheduling write, active plugin")
	//Log.Info(s.activePlugin)
	return dp.scheduler.ScheduleWrite(data, stream, dp.Streams)
}

func (dp *QPartsDataplane) AddListenStream(id uint64, ssqc *qpnet.SingleStreamQUICConn) error {
	dp.Streams[id] = &QPartsDataplaneStream{
		ssqc:  ssqc,
		Queue: qpnet.NewQueue[DataAssignment](),
	}
	qplogging.Log.Debug("Added listen stream")

	return nil
}

func (dp *QPartsDataplane) writeLoop() error {

	var wg sync.WaitGroup

	for streamId, stream := range dp.Streams {
		wg.Add(1)
		qplogging.Log.Info("Starting write loop for stream ", streamId)
		go func(streamId uint64, stream *QPartsDataplaneStream) {
			for {
				dataAssignment := stream.Queue.Get()
				compl := dp.completionStore.GetOrCreateSendingSequenceCompletion(dataAssignment.StreamId, dataAssignment.SequenceId, dataAssignment.NumParts, dataAssignment.SequenceSize)
				// fmt.Println("Sending data packet: ", i)
				partsDatapacket := qpproto.NewQPartsDataplanePacket()
				partsDatapacket.Flags = PARTS_MSG_DATA
				partsDatapacket.StreamId = dataAssignment.StreamId
				qplogging.Log.Info("STREAMID ", partsDatapacket.StreamId)
				partsDatapacket.SequenceId = compl.SequenceId
				partsDatapacket.PartId = dataAssignment.PartId
				partsDatapacket.SequenceSize = compl.SequenceSize
				partsDatapacket.NumParts = dataAssignment.NumParts

				partsDatapacket.PartSize = uint64(len(dataAssignment.Data))
				partsDatapacket.Encode()

				n, err := dataAssignment.DataplaneStream.ssqc.WriteAll(partsDatapacket.Data)
				if err != nil {
					qplogging.Log.Warn("Error writing data packet: ", err)
					continue
				}

				if n <= 0 {
					qplogging.Log.Warn("No data sent on dataconn")
					continue
				}
				// qplogging.Log.Debug("Sent data packet: ", n)

				n, err = dataAssignment.DataplaneStream.ssqc.WriteAll(dataAssignment.Data)
				if err != nil {
					qplogging.Log.Warn("Error writing data packet: ", err)
					continue
				}
				if n <= 0 {
					qplogging.Log.Warn("No data sent on dataconn")
					continue
				}
				fmt.Println("Sent data packet: ", n)
				qplogging.Log.Debug("Compl: ", compl.CompletedParts, " / ", compl.Parts)
				completed := compl.AddSendingPart(int(dataAssignment.PartId), dataAssignment.Data)
				if completed {
					dp.completionStore.RemoveCompletion(compl.SequenceId)
					fmt.Println("Sent all parts for sequence ", compl.SequenceId)
					fmt.Printf("Triggering completion %p\n", compl)
					compl.CompleteChan <- true
				}

			}
			wg.Done()
		}(streamId, stream)
	}

	wg.Wait()
	return nil
}

/*func (dp *QPartsDataplane) WriteDataForStream(data []byte, id uint64) (int, error) {
	qplogging.Log.Debug("Writing to stream ", id)

	compl := dp.completionStore.NewSequenceCompletionFromData(id, data)
	dp.completionStore.AddCompletion(compl)

	fmt.Printf("Waiting for completion %p\n", compl)
	<-compl.CompleteChan
	sentBytes := len(data)
	fmt.Println("Sent ", sentBytes, " bytes")
	return sentBytes, nil
}*/

func (dp *QPartsDataplane) WriteForStream(schedulingDecision *SchedulingDecision, id uint64) (int, error) {
	qplogging.Log.Debug("Writing to stream ", id)

	compl := dp.completionStore.NewSequenceCompletionFromSchedulingDecision(id, schedulingDecision)
	dp.completionStore.AddCompletion(compl)

	// atomic int
	sentBytes := 0
	// TODO: Move to queue based approach for all streams?
	for i, dataAssignment := range schedulingDecision.Assignments {
		//wg.Add(1)
		//go func(dataAssignment DataAssignment, i int) {

		// qplogging.Log.Debugf("Copying data %x from %d to %d \n", sha256.Sum256(dataAssignment.Data), i, i+len(dataAssignment.Data))
		// qplogging.Log.Debugf("Sent %x on Stream %d for id %d\n", sha256.Sum256(dataAssignment.Data), id, int(partsDatapacket.PartId))
		//	sentBytes += len(dataAssignment.Data)
		//	wg.Done()
		//}(dataAssignment, i)
		dataAssignment.NumParts = uint64(len(schedulingDecision.Assignments))
		dataAssignment.SequenceId = compl.SequenceId
		dataAssignment.PartId = uint64(i)
		dataAssignment.SequenceSize = compl.SequenceSize
		dataAssignment.StreamId = id

		dataAssignment.DataplaneStream.Queue.Add(dataAssignment)
		qplogging.Log.Debug("Added data assignment to queue")

		sentBytes += len(dataAssignment.Data)
	}

	// dp.completionStore.RemoveCompletion(compl.SequenceId)

	// wg.Wait()
	fmt.Printf("Waiting for completion %p\n", compl)
	<-compl.CompleteChan
	fmt.Println("Sent ", sentBytes, " bytes")
	return sentBytes, nil
}

func (dp *QPartsDataplane) readLoop() error {
	// TODO: Make it capable of adding/removing streams
	qplogging.Log.Debug("Starting read loop")
	var wg sync.WaitGroup
	for streamId, stream := range dp.Streams {
		wg.Add(1)
		go func(streamId uint64, stream *QPartsDataplaneStream) {
			for {
				partsDatapacket := qpproto.NewQPartsDataplanePacket()

				// qplogging.Log.Debug(partsDatapacket)
				qplogging.Log.Debug("Reading ", len(partsDatapacket.Data), " bytes on  stream ", streamId)
				n, err := stream.ssqc.ReadAll(partsDatapacket.Data)
				if err != nil {
					qplogging.Log.Warn("Error reading data packet: ", err)
					continue
				}
				if n <= 0 {
					qplogging.Log.Warn("No data received on dataconn")
					continue
				}
				partsDatapacket.Decode()
				qplogging.Log.Info("STREAMID ", partsDatapacket.StreamId)

				compl := dp.completionStore.GetOrCreateSequenceCompletion(partsDatapacket.StreamId, partsDatapacket.SequenceId, partsDatapacket.NumParts, partsDatapacket.SequenceSize)

				//qplogging.Log.Debugf("Received compl %p %d %d %d %d %d\n\n", compl, compl.StreamId, compl.SequenceId, compl.Parts, compl.SequenceSize, compl.CompletedParts)
				qplogging.Log.Debug("With size ", partsDatapacket.PartSize)

				data := make([]byte, partsDatapacket.PartSize)

				n, err = stream.ssqc.ReadAll(data)
				if err != nil {
					qplogging.Log.Warn("Error reading data packet: ", err)
					continue
				}
				if n <= 0 {
					qplogging.Log.Warn("No data received on dataconn")
					continue
				}

				isComplete := compl.AddPart(int(partsDatapacket.PartId), data)
				qplogging.Log.Debug("Having ", compl.CompletedParts, " / ", compl.Parts, " parts for sequence ", compl.SequenceId)
				// qplogging.Log.Debugf("Received %x on stream %d for id %d \n", sha256.Sum256(data), streamId, int(partsDatapacket.PartId))
				// time.Sleep(3 * time.Second)

				if isComplete {

					// TODO: Access QPARTSStream Here
					s := dp.QPartsStreams[partsDatapacket.StreamId]
					dp.completionStore.RemoveCompletion(partsDatapacket.SequenceId)
					qplogging.Log.Info("Trying to access stream ", partsDatapacket.StreamId)
					//qplogging.Log.Info("compl ", compl)
					//qplogging.Log.Info("stream ", s)
					s.ReadBuffer.Append(compl.Data)
					qplogging.Log.Debug("Sequence complete: ", compl.SequenceId)
				}

			}
		}(streamId, stream)
	}

	wg.Wait()
	return nil
}

// FindSimilarPaths finds paths with similar PacketLossEvents/Throughput or CwndDecreaseEvents/Throughput ratios
func FindSimilarPaths(paths []qpmetrics.PathMetrics, tolerance float64) [][]qpmetrics.PathMetrics {
	var result [][]qpmetrics.PathMetrics

	// Helper function to calculate similarity
	isSimilar := func(a, b float64, tol float64) bool {
		fmt.Println(a, b, tol, math.Abs(a-b))
		return math.Abs(a-b) <= tol
	}

	for i := 0; i < len(paths); i++ {
		for j := i + 1; j < len(paths); j++ {
			// Avoid division by zero
			if paths[i].Throughput == 0 || paths[j].Throughput == 0 {
				continue
			}

			if paths[i].Packetloss == 0 || paths[j].Packetloss == 0 {
				continue
			}

			// Calculate ratios
			//qplogging.Log.Debug("-----------------------------------------------------")
			//qplogging.Log.Debugf("Path %d: Throughput: %d, Packetloss: %d", paths[i].PathId, paths[i].Throughput, paths[i].Packetloss)
			//qplogging.Log.Debugf("Path %d: Throughput: %d, Packetloss: %d", paths[j].PathId, paths[j].Throughput, paths[j].Packetloss)
			//qplogging.Log.Debug("-----------------------------------------------------")
			ratio1Loss := float64(paths[i].Packetloss) / float64(paths[i].Throughput)
			ratio2Loss := float64(paths[j].Packetloss) / float64(paths[j].Throughput)
			ratio1Cwnd := float64(paths[i].CwndDecreaseEvents) / float64(paths[i].Throughput)
			ratio2Cwnd := float64(paths[j].CwndDecreaseEvents) / float64(paths[j].Throughput)

			// Check similarity
			if isSimilar(ratio1Loss, ratio2Loss, tolerance) || isSimilar(ratio1Cwnd, ratio2Cwnd, tolerance) {
				result = append(result, []qpmetrics.PathMetrics{paths[i], paths[j]})
			}
		}
	}

	return result
}
