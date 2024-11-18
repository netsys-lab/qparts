package qparts

import (
	"crypto/sha256"
	"fmt"
	"math/rand"
	"sync"

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
}

type QPartsDataplaneStream struct {
	ssqc *qpnet.SingleStreamQUICConn
	// path      snet.DataplanePath
	PartsPath *qpscion.QPartsPath
}

func NewQPartsDataplane(scheduler *Scheduler, partsStreams map[uint64]*PartsStream) *QPartsDataplane {
	return &QPartsDataplane{
		Streams:         make(map[uint64]*QPartsDataplaneStream),
		scheduler:       scheduler,
		QPartsStreams:   partsStreams,
		completionStore: NewCompletionStore(),
	}
}

func (dp *QPartsDataplane) GetStream(id uint64) *QPartsDataplaneStream {
	return dp.Streams[id]
}

func (dp *QPartsDataplane) AddDialStream(id uint64, ssqc *qpnet.SingleStreamQUICConn, path *qpscion.QPartsPath) error {
	dp.Streams[id] = &QPartsDataplaneStream{
		ssqc:      ssqc,
		PartsPath: path,
	}
	fmt.Println("Added dial stream")

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
		ssqc: ssqc,
	}
	fmt.Println("Added listen stream")

	return nil
}

// Function to generate a random byte array of a given size
func generateRandomBytes(size int) ([]byte, error) {
	bytes := make([]byte, size)
	_, err := rand.Read(bytes)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

func (dp *QPartsDataplane) WriteForStream(schedulingDecision *SchedulingDecision, id uint64) (int, error) {
	fmt.Println("Writing to stream ", id)
	var wg sync.WaitGroup

	compl := dp.completionStore.NewSequenceCompletionFromSchedulingDecision(id, schedulingDecision)
	dp.completionStore.AddCompletion(compl)

	// TODO: Move to queue based approach for all streams?
	for i, dataAssignment := range schedulingDecision.Assignments {
		wg.Add(1)
		go func(dataAssignment DataAssignment, i int) {
			partsDatapacket := qpproto.NewQPartsDataplanePacket()
			partsDatapacket.Flags = PARTS_MSG_DATA
			partsDatapacket.StreamId = id
			partsDatapacket.SequenceId = compl.SequenceId
			partsDatapacket.PartId = uint64(i)
			partsDatapacket.SequenceSize = compl.SequenceSize
			partsDatapacket.NumParts = uint64(len(schedulingDecision.Assignments))

			partsDatapacket.PartSize = uint64(len(dataAssignment.Data))
			partsDatapacket.Encode()

			n, err := dataAssignment.DataplaneStream.ssqc.WriteAll(partsDatapacket.Data)
			if err != nil {
				panic(err)
			}

			if n <= 0 {
				panic("No data sent")
			}
			// fmt.Println("Sent data packet: ", n)

			n, err = dataAssignment.DataplaneStream.ssqc.WriteAll(dataAssignment.Data)
			if err != nil {
				panic(err)
			}
			if n <= 0 {
				panic("No data sent")
			}

			fmt.Printf("Sent %x on Stream %d\n", sha256.Sum256(dataAssignment.Data), id)
			wg.Done()
		}(dataAssignment, i)
	}

	dp.completionStore.RemoveCompletion(compl.SequenceId)

	wg.Wait()
	return 0, nil

	/*for streamId, stream := range dp.Streams {
		wg.Add(1)
		go func(streamId uint64, stream *QPartsDataplaneStream) {

			partsDatapacket := NewQPartsDataplanePacket()
			partsDatapacket.Flags = PARTS_MSG_DATA
			partsDatapacket.StreamId = id
			partsDatapacket.PartId = 1
			partsDatapacket.FrameId = 1

			data, _ := generateRandomBytes(1000000)
			fmt.Printf("%x\n", sha256.Sum256(data))

			partsDatapacket.FrameSize = uint64(len(data))
			partsDatapacket.Encode()

			n, err := stream.ssqc.WriteAll(partsDatapacket.Data)
			if err != nil {
				panic(err)
			}

			if n <= 0 {
				panic("No data sent")
			}
			fmt.Println("Sent data packet: ", n)

			n, err = stream.ssqc.WriteAll(data)
			if err != nil {
				panic(err)
			}
			if n <= 0 {
				panic("No data sent")
			}

			fmt.Println("sENT: ", partsDatapacket)
			fmt.Println("On stream ", streamId)

		}(streamId, stream)
	}*/

}

func (dp *QPartsDataplane) readLoop() error {
	// TODO: Make it capable of adding/removing streams
	fmt.Println("Starting read loop")
	var wg sync.WaitGroup
	for streamId, stream := range dp.Streams {
		wg.Add(1)
		go func(streamId uint64, stream *QPartsDataplaneStream) {

			for {
				partsDatapacket := qpproto.NewQPartsDataplanePacket()
				n, err := stream.ssqc.ReadAll(partsDatapacket.Data)
				if err != nil {
					panic(err)
				}
				if n <= 0 {
					panic("No data received")
				}

				partsDatapacket.Decode()

				compl := dp.completionStore.GetOrCreateSequenceCompletion(partsDatapacket.StreamId, partsDatapacket.SequenceId, partsDatapacket.NumParts, partsDatapacket.SequenceSize)

				// fmt.Println("Received: ", partsDatapacket)
				// fmt.Println("On stream ", streamId)

				data := make([]byte, partsDatapacket.PartSize)
				n, err = stream.ssqc.ReadAll(data)
				if err != nil {
					panic(err)
				}
				if n <= 0 {
					panic("No data received")
				}

				compl.AddPart(int(partsDatapacket.PartId), data)
				fmt.Println("Received: ", partsDatapacket)
				fmt.Println(compl.IsComplete())
				fmt.Println("Having ", compl.CompletedParts, " / ", compl.Parts)
				if compl.IsComplete() {
					// TODO: Access QPARTSStream Here
					s := dp.QPartsStreams[partsDatapacket.StreamId]
					s.ReadBuffer.Append(compl.Data)
					dp.completionStore.RemoveCompletion(partsDatapacket.SequenceId)
				}

				fmt.Printf("Received %x on stream %d\n", sha256.Sum256(data), streamId)
			}
		}(streamId, stream)
	}

	wg.Wait()
	return nil
}
