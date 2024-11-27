package qparts

import (
	"fmt"
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

func (sr *SchedulerECMP) PathSelectionAfterCongestionEvent(preference uint32, event *qpmetrics.CongestionEvent, availablePaths []qpscion.QPartsPath, pathsInUse []qpscion.QPartsPath) []qpscion.QPartsPath {
	return nil
}

func (sr *SchedulerECMP) PathSelectionForProbing(preference uint32, availablePaths []qpscion.QPartsPath, pathsInUse []qpscion.QPartsPath) []qpscion.QPartsPath {
	return availablePaths
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

func (sr *SchedulerECMP) InitialPathSelection(preference uint32, paths []qpscion.QPartsPath) ([]qpscion.QPartsPath, error) {
	// Implement the logic for initial path selection here
	qplogging.Log.Debug("Selecting paths for ECMP initially")

	/*tpaths := []qpscion.QPartsPath{
		{Id: 1, Interfaces: []string{"A", "B", "C"}, MTU: 1452, Hops: 3},
		{Id: 2, Interfaces: []string{"A", "D", "E"}, MTU: 1472, Hops: 5},
		{Id: 3, Interfaces: []string{"A", "F"}, MTU: 1472, Hops: 2},
		{Id: 4, Interfaces: []string{"A", "G", "H"}, MTU: 1472, Hops: 5},
		{Id: 5, Interfaces: []string{"A", "I", "J"}, MTU: 1452, Hops: 6},
	}*/

	if len(paths) == 0 {
		return nil, fmt.Errorf("no paths available for initial path selection")
	}

	lowLatency, highThroughput := selectPaths(paths)
	qplogging.Log.Debug("Low Latency Path:", lowLatency.Sorter)

	returnPaths := make([]qpscion.QPartsPath, 0)
	returnPaths = append(returnPaths, lowLatency)

	if len(highThroughput) > 0 {
		for i, p := range highThroughput {
			qplogging.Log.Debugf("High Throughput path %d: %s", i, p.Sorter)
			returnPaths = append(returnPaths, p)
		}
	}

	// qplogging.Log.Debug("High Throughput Paths:", highThroughput[0].Id, highThroughput[1].Id)
	qplogging.Log.Debug("Selected ", len(returnPaths), " paths for initial path selection")
	// qplogging.Log.Debug(paths[:3])
	return returnPaths, nil
	//return paths[:3], nil
}

// selectPaths selects paths for low latency and high throughput
func selectPaths(paths []qpscion.QPartsPath) (lowLatency qpscion.QPartsPath, highThroughput []qpscion.QPartsPath) {

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
func findMinHops(paths []qpscion.QPartsPath) int {
	minHops := paths[0].Hops
	for _, p := range paths {
		if p.Hops < minHops {
			minHops = p.Hops
		}
	}
	return minHops
}

// filterPathsByHops filters paths with the given hop count
func filterPathsByHops(paths []qpscion.QPartsPath, hops int) []qpscion.QPartsPath {
	var filtered []qpscion.QPartsPath
	for _, p := range paths {
		if p.Hops == hops {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// findMaxMTU finds the maximum MTU among paths
func findMaxMTU(paths []qpscion.QPartsPath) int {
	maxMTU := paths[0].MTU
	for _, p := range paths {
		if p.MTU > maxMTU {
			maxMTU = p.MTU
		}
	}
	return maxMTU
}

// filterPathsByMTU filters paths with the given MTU and excludes a specific path
func filterPathsByMTU(paths []qpscion.QPartsPath, mtu int, exclude qpscion.QPartsPath) []qpscion.QPartsPath {
	var filtered []qpscion.QPartsPath
	for _, p := range paths {
		if p.MTU == mtu && p.Id != exclude.Id {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// excludePath excludes specific paths from a list
func excludePath(paths []qpscion.QPartsPath, exclude []qpscion.QPartsPath) []qpscion.QPartsPath {
	excludeMap := make(map[uint64]bool)
	for _, p := range exclude {
		excludeMap[p.Id] = true
	}

	var filtered []qpscion.QPartsPath
	for _, p := range paths {
		if !excludeMap[p.Id] {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// selectRandomPath selects a random path from a list
func selectRandomPath(paths []qpscion.QPartsPath) qpscion.QPartsPath {
	return paths[rand.Intn(len(paths))]
}

// shufflePaths shuffles the order of paths
func shufflePaths(paths []qpscion.QPartsPath) {
	rand.Shuffle(len(paths), func(i, j int) {
		paths[i], paths[j] = paths[j], paths[i]
	})
}

// selectTopN selects the top N paths from a list
func selectTopN(paths []qpscion.QPartsPath, n int) []qpscion.QPartsPath {
	if n > len(paths) {
		n = len(paths)
	}
	return paths[:n]
}
