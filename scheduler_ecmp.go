package qparts

import (
	"fmt"

	"github.com/netsys-lab/qparts/pkg/qpnet"
	"github.com/netsys-lab/qparts/pkg/qpscion"
)

// Ensure that SchedulerSinglePath implements the IScheduler interface
var _ IScheduler = (*SchedulerECMP)(nil)

type SchedulerECMP struct {
	index int
}

func NewSchedulerECMP() *SchedulerECMP {
	return &SchedulerECMP{
		index: 0,
	}
}

func (sr *SchedulerECMP) OnCongestionEvent(event *qpnet.CongestionEvent) error {
	return nil
}

func (sr *SchedulerECMP) ScheduleWrite(data []byte, stream *PartsStream, dpStreams map[uint64]*QPartsDataplaneStream) SchedulingDecision {
	// Log.Info(stream.Conn.remote)
	/*s := stream.conn.remote.String()

	rem, ok := state.Remotes[s]
	if !ok {
		Log.Info("Adding remote")
		state.AddRemote(stream.conn.remote)
		rem = state.Remotes[s]
	}*/

	// Distribute data over all available dpStreams

	distribution := splitBytes(data, len(dpStreams))

	assignments := make([]DataAssignment, len(dpStreams))
	index := 0
	for _, p := range dpStreams {
		da := DataAssignment{
			// Path: rem.Paths[sr.index],
			DataplaneStream: p, // dpStreams[0],
			Data:            distribution[index],
		}
		assignments[index] = da
		index++
	}

	// fmt.Println("Assignments: ", assignments)

	return SchedulingDecision{
		Assignments: assignments,
	}
}

// splitBytes splits a slice of bytes into n parts
func splitBytes(data []byte, n int) [][]byte {
	if n <= 0 {
		return nil // Invalid number of parts
	}

	// Handle the case where n is greater than the length of the data
	if n > len(data) {
		n = len(data)
	}

	// Determine the size of each part
	partSize := len(data) / n
	remainder := len(data) % n

	var parts [][]byte
	start := 0

	for i := 0; i < n; i++ {
		// Calculate the size for this part
		size := partSize
		if i < remainder {
			size++ // Distribute the remainder across the first few parts
		}

		// Append the part to the result
		end := start + size
		parts = append(parts, data[start:end])
		start = end
	}

	return parts
}

func (sr *SchedulerECMP) InitialPathSelection(preference uint32, paths []qpscion.QPartsPath) ([]qpscion.QPartsPath, error) {
	// Implement the logic for initial path selection here
	fmt.Println("Selecting paths for ECMP")
	fmt.Println(paths[:1])
	return paths[:1], nil
}
