package qparts

import (
	"math/rand"

	"github.com/netsys-lab/qparts/pkg/qplogging"
	"github.com/netsys-lab/qparts/pkg/qpmetrics"
	"github.com/netsys-lab/qparts/pkg/qpscion"
)

// Ensure that SchedulerSinglePath implements the IScheduler interface
var _ SchedulerPlugin = (*SchedulerECMP)(nil)

type SchedulerECMP struct {
	index int
}

func NewSchedulerECMP() *SchedulerECMP {
	return &SchedulerECMP{
		index: 0,
	}
}

func (sr *SchedulerECMP) OnNewStream(remote string, preference uint32) {

}

func (sr *SchedulerECMP) InitialPathSelection(remote string, preference uint32) {
	// Implement the logic for initial path selection here
	qplogging.Log.Debug("Selecting paths for ECMP initially")

	lowLatency, highThroughput := selectLowLatencyAndHighThroughputPaths(remote)
	qplogging.Log.Debug("Low Latency Path:", lowLatency.Sorter)

	r := qpmetrics.State.Remotes[remote]
	r[lowLatency.Id].Path.Pref = qpscion.QPARTS_PATH_PREF_LOW_LATENCY
	r[lowLatency.Id].Path.State = qpscion.QPARTS_PATH_STATE_ACTIVE

	for _, p := range highThroughput {
		r[p.Id].Path.Pref = qpscion.QPARTS_PATH_PREF_HIGH_THROUGHPUT
		r[p.Id].Path.State = qpscion.QPARTS_PATH_STATE_ACTIVE
	}

	// qplogging.Log.Debug("High Throughput Paths:", highThroughput[0].Id, highThroughput[1].Id)
	qplogging.Log.Debug("Selected ", len(highThroughput)+1, " paths for initial path selection")
	// qplogging.Log.Debug(paths[:3])

}

func (sr *SchedulerECMP) OnPathProbeInterval(remote string) {

}
func (sr *SchedulerECMP) OnCongestionEvent(event *qpmetrics.CongestionEvent) {

}
func (sr *SchedulerECMP) OnPathDown(remote string, pathId uint64) {

}
func (sr *SchedulerECMP) OnPathAdd(remote string, pathId uint64) {

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

	var s *QPartsDataplaneStream
	for _, p := range dpStreams {
		s = p
		break
	}

	if len(data) <= 1000 {
		da := DataAssignment{
			DataplaneStream: s,
			Data:            data,
		}
		return SchedulingDecision{
			Assignments: []DataAssignment{da},
		}
	}

	distribution := splitBytes(data, len(dpStreams))

	assignments := make([]DataAssignment, 0)
	index := 0

	// Just iterate over streams and give each stream a size, then continue

	for _, p := range dpStreams {

		// TODO: SPlit this by dpStream.OptimalPartSize
		for j := 0; j < 5; j++ {
			partSize := len(distribution[index]) / 5
			max := min((j+1)*partSize, len(distribution[index]))

			if j == 4 {
				max = len(distribution[index])
			}

			da := DataAssignment{
				// Path: rem.Paths[sr.index],
				DataplaneStream: p, // dpStreams[0],
				Data:            distribution[index][j*partSize : max],
			}
			assignments = append(assignments, da)
		}

		index++
	}

	// qplogging.Log.Debug("Assignments: ", assignments)

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

	for i, p := range parts {
		qplogging.Log.Debugf("Part %d: has len %d", i, len(p))
	}

	return parts
}

// selectPaths selects paths for low latency and high throughput
func selectLowLatencyAndHighThroughputPaths(remote string) (lowLatency *qpscion.QPartsPath, highThroughput []*qpscion.QPartsPath) {

	paths := qpmetrics.State.ToPathSlice(remote)

	// Use the same path for everything
	if len(paths) == 1 {
		return paths[0], nil
	}

	if len(paths) == 2 {
		return paths[0], paths[1:]
	}

	// Step 2: Identify the low-latency path (shortest hops)
	minHops := findMinHops(paths)
	lowLatencyCandidates := filterPathsByHops(paths, minHops)
	lowLatency = selectRandomPath(lowLatencyCandidates)

	// Step 3: Identify high-throughput paths (highest MTU)
	maxMTU := findMaxMTU(paths)
	highThroughputCandidates := filterPathsByMTU(paths, maxMTU, lowLatency)
	shufflePaths(highThroughputCandidates)

	// Select two high-throughput paths
	highThroughput = selectTopN(highThroughputCandidates, 2)

	// Step 4: Ensure fallback for fewer high-throughput candidates
	if len(highThroughput) < 2 {
		additionalCandidates := excludePath(paths, append(highThroughput, lowLatency))
		shufflePaths(additionalCandidates)
		highThroughput = append(highThroughput, selectTopN(additionalCandidates, 2-len(highThroughput))...)
	}

	return lowLatency, highThroughput
}

// findMinHops finds the minimum hops among paths
func findMinHops(paths []*qpscion.QPartsPath) int {
	minHops := paths[0].Hops
	for _, p := range paths {
		if p.Hops < minHops {
			minHops = p.Hops
		}
	}
	return minHops
}

// filterPathsByHops filters paths with the given hop count
func filterPathsByHops(paths []*qpscion.QPartsPath, hops int) []*qpscion.QPartsPath {
	var filtered []*qpscion.QPartsPath
	for _, p := range paths {
		if p.Hops == hops {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// findMaxMTU finds the maximum MTU among paths
func findMaxMTU(paths []*qpscion.QPartsPath) int {
	maxMTU := paths[0].MTU
	for _, p := range paths {
		if p.MTU > maxMTU {
			maxMTU = p.MTU
		}
	}
	return maxMTU
}

// filterPathsByMTU filters paths with the given MTU and excludes a specific path
func filterPathsByMTU(paths []*qpscion.QPartsPath, mtu int, exclude *qpscion.QPartsPath) []*qpscion.QPartsPath {
	var filtered []*qpscion.QPartsPath
	for _, p := range paths {
		if p.MTU == mtu && p.Id != exclude.Id {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// excludePath excludes specific paths from a list
func excludePath(paths []*qpscion.QPartsPath, exclude []*qpscion.QPartsPath) []*qpscion.QPartsPath {
	excludeMap := make(map[uint64]bool)
	for _, p := range exclude {
		excludeMap[p.Id] = true
	}

	var filtered []*qpscion.QPartsPath
	for _, p := range paths {
		if !excludeMap[p.Id] {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// selectRandomPath selects a random path from a list
func selectRandomPath(paths []*qpscion.QPartsPath) *qpscion.QPartsPath {
	return paths[rand.Intn(len(paths))]
}

// shufflePaths shuffles the order of paths
func shufflePaths(paths []*qpscion.QPartsPath) {
	rand.Shuffle(len(paths), func(i, j int) {
		paths[i], paths[j] = paths[j], paths[i]
	})
}

// selectTopN selects the top N paths from a list
func selectTopN(paths []*qpscion.QPartsPath, n int) []*qpscion.QPartsPath {
	if n > len(paths) {
		n = len(paths)
	}
	return paths[:n]
}
