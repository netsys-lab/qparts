package qpmetrics

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/netsys-lab/qparts/pkg/qplogging"
)

type PathMetrics struct {
	RTT                uint64
	Packetloss         uint64
	Throughput         uint64
	PathId             uint64
	CwndDecreaseEvents uint64
	PacketLossEvents   uint64
	Sorter             string
}

func NewPathMetrics() *PathMetrics {
	return &PathMetrics{}
}

type NetworkState struct {
	sync.Mutex
	Remotes           map[string]map[uint64]*PathMetrics
	OnCongestionEvent []func(event *CongestionEvent) error
}

var State *NetworkState

func init() {
	State = &NetworkState{
		Remotes: make(map[string]map[uint64]*PathMetrics),
		/*OnCongestionEvent: []func(event *CongestionEvent) error{func(event *CongestionEvent) error {
			qplogging.Log.Warn("No congestion event handler set")
			return nil
		}},*/
	}
}

func (n *NetworkState) StartMeasurements() {
	go n.runDetectionTicker()
}

func (n *NetworkState) runDetectionTicker() {
	ticker := time.NewTicker(500 * time.Millisecond)

	for range ticker.C {
		qplogging.Log.Debug("Running stats check")
		n.DetectCongestion()

		for _, rem := range n.Remotes {
			for _, path := range rem {
				// tr := float64(path.Throughput*8*2) / 1024 / 1024
				// qplogging.Log.Debugf("Path %d: RTT: %d, Packetloss: %d, Throughput: %f Mbit/s, CwndDecreaseEvents: %d, PacketLossEvents: %d", path.PathId, path.RTT, path.Packetloss, tr, path.CwndDecreaseEvents, path.PacketLossEvents)
				path.Throughput = 0
				path.Packetloss = 0
			}
		}

	}
}

func (n *NetworkState) DetectCongestion() {
	allPaths := make([]PathMetrics, 0)
	for _, rem := range n.Remotes {
		for _, path := range rem {

			if path.PathId == 0 || path.Throughput == 0 {
				continue
			}

			allPaths = append(allPaths, *path)
		}
	}

	simPaths := FindSimilarPaths(allPaths, 0.1)
	fmt.Println("Similar Paths:")
	for _, pair := range simPaths {
		fmt.Printf("Pair: PathId %s and PathId %s\n", pair[0].Sorter, pair[1].Sorter)
	}
}

func (n *NetworkState) AddCongestionEventHandler(handler func(event *CongestionEvent) error) {
	n.OnCongestionEvent = append(n.OnCongestionEvent, handler)
}

func (n *NetworkState) AddPathMetrics(remote string, pathId uint64, metrics *PathMetrics) {

	if pathId == 0 {
		panic("PathId cannot be 0")
	}

	if _, ok := n.Remotes[remote]; !ok {
		n.Remotes[remote] = make(map[uint64]*PathMetrics)
	}
	n.Remotes[remote][pathId] = metrics
}

func (n *NetworkState) AddPacketLoss(remote string, pathId uint64, loss uint64, sorter string) {

	if pathId == 0 {
		panic("PathId cannot be 0")
	}

	if _, ok := n.Remotes[remote]; !ok {
		n.Remotes[remote] = make(map[uint64]*PathMetrics)
	}

	if metrics, ok := n.Remotes[remote][pathId]; !ok {
		metrics = NewPathMetrics()
		metrics.PathId = pathId
		n.Remotes[remote][pathId] = metrics
		n.Remotes[remote][pathId].Sorter = sorter
		qplogging.Log.Debugf("Added path %s for remote %s", sorter, remote)
	}

	n.Remotes[remote][pathId].Packetloss += loss
}

func (n *NetworkState) AddThroughput(remote string, pathId uint64, bytecount uint64, sorter string) {

	if pathId == 0 {
		panic("PathId cannot be 0")
	}

	if _, ok := n.Remotes[remote]; !ok {
		n.Remotes[remote] = make(map[uint64]*PathMetrics)
	}

	if metrics, ok := n.Remotes[remote][pathId]; !ok {
		metrics = NewPathMetrics()
		metrics.PathId = pathId
		n.Remotes[remote][pathId] = metrics
		n.Remotes[remote][pathId].Sorter = sorter
		qplogging.Log.Debugf("Added path %s for remote %s", sorter, remote)
	}

	n.Remotes[remote][pathId].Throughput += bytecount
}

func (n *NetworkState) GetPathMetrics(remote string, pathId uint64) *PathMetrics {
	if _, ok := n.Remotes[remote]; !ok {
		return nil
	}
	return n.Remotes[remote][pathId]
}

/**
// Example usage
func main() {
	paths := []PathMetrics{
		{RTT: 10, Packetloss: 5, Throughput: 100, PathId: 1, CwndDecreaseEvents: 50, PacketLossEvents: 30},
		{RTT: 15, Packetloss: 4, Throughput: 95, PathId: 2, CwndDecreaseEvents: 50, PacketLossEvents: 30},
		{RTT: 20, Packetloss: 6, Throughput: 2000, PathId: 3, CwndDecreaseEvents: 1, PacketLossEvents: 1},
	}

	// Tolerance for similarity
	tolerance := 0.1

	similarPaths := FindSimilarPaths(paths, tolerance)
	fmt.Println("Similar Paths:")
	for _, pair := range similarPaths {
		fmt.Printf("Pair: PathId %d and PathId %d\n", pair[0].PathId, pair[1].PathId)
	}
}
	**/

// FindSimilarPaths finds paths with similar PacketLossEvents/Throughput or CwndDecreaseEvents/Throughput ratios
func FindSimilarPaths(paths []PathMetrics, tolerance float64) [][]PathMetrics {
	var result [][]PathMetrics

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
			// ratio1Cwnd := float64(paths[i].CwndDecreaseEvents) / float64(paths[i].Throughput)
			// ratio2Cwnd := float64(paths[j].CwndDecreaseEvents) / float64(paths[j].Throughput)

			// Check similarity
			if isSimilar(ratio1Loss, ratio2Loss, tolerance) /*|| isSimilar(ratio1Cwnd, ratio2Cwnd, tolerance)*/ {
				result = append(result, []PathMetrics{paths[i], paths[j]})
			}
		}
	}

	return result
}
