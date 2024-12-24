package qpmetrics

import (
	"sync"

	"github.com/netsys-lab/qparts/pkg/qpscion"
	"github.com/scionproto/scion/pkg/snet"
)

/**
* Add qpscion.QPartsPath to metrics
* Extend NetworkState with keeping all the path information, including path state
* Let scheduler work on network state to perform path selection
* Add OnPathDown event to scheduler
* Add OnPathAdd event to scheduler
* Do background path monitoring to see if new paths are available to all active peers
* Whenever path set is changed, dataplane needs to be updated -> this may be a bit difficult
* I think all these OnPath and stuff should be somewhere in the dataplane, so the dataplane runs timers to do stuff on the NetworkState object
**/

type PathMetrics struct {
	Path               *qpscion.QPartsPath
	RTT                uint64
	Packetloss         uint64
	Throughput         uint64
	CwndDecreaseEvents uint64
	PacketLossEvents   uint64
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

func (n *NetworkState) ActivePaths(remote string) []*qpscion.QPartsPath {
	var paths []*qpscion.QPartsPath
	for _, path := range n.Remotes[remote] {

		if path.Path.State == qpscion.QPARTS_PATH_STATE_ACTIVE {
			paths = append(paths, path.Path)
		}
	}

	return paths
}

func (n *NetworkState) ToPathSlice(remote string) []*qpscion.QPartsPath {
	var paths []*qpscion.QPartsPath
	for _, path := range n.Remotes[remote] {
		paths = append(paths, path.Path)
	}

	return paths
}

func (n *NetworkState) AddRemote(remote *snet.UDPAddr) error {
	n.Lock()
	defer n.Unlock()
	r := remote.String()
	if _, ok := n.Remotes[r]; !ok {
		n.Remotes[r] = make(map[uint64]*PathMetrics)
	}

	paths, err := qpscion.QueryPaths(remote.IA)
	if err != nil {
	}

	for _, path := range paths {
		n.Remotes[remote.String()][path.Id] = &PathMetrics{
			Path: &path,
		}
	}

	return nil
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

	n.Remotes[remote][pathId].Packetloss += loss
}

func (n *NetworkState) AddConnStateRecovery(remote string, pathId uint64, loss uint64, sorter string) {

	if pathId == 0 {
		panic("PathId cannot be 0")
	}

	if _, ok := n.Remotes[remote]; !ok {
		n.Remotes[remote] = make(map[uint64]*PathMetrics)
	}

	n.Remotes[remote][pathId].CwndDecreaseEvents += loss
}

func (n *NetworkState) AddThroughput(remote string, pathId uint64, bytecount uint64, sorter string) {

	if pathId == 0 {
		panic("PathId cannot be 0")
	}

	if _, ok := n.Remotes[remote]; !ok {
		n.Remotes[remote] = make(map[uint64]*PathMetrics)
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
