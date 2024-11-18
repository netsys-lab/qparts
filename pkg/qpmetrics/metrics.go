package qpmetrics

type PathMetrics struct {
	RTT        uint64
	Packetloss uint64
	Throughput uint64
	PathId     uint64
}

func NewPathMetrics() *PathMetrics {
	return &PathMetrics{}
}

type NetworkState struct {
	Remotes map[string]map[uint64]*PathMetrics
}

var State *NetworkState

func init() {
	State = &NetworkState{
		Remotes: make(map[string]map[uint64]*PathMetrics),
	}
}

func (n *NetworkState) AddPathMetrics(remote string, pathId uint64, metrics *PathMetrics) {
	if _, ok := n.Remotes[remote]; !ok {
		n.Remotes[remote] = make(map[uint64]*PathMetrics)
	}
	n.Remotes[remote][pathId] = metrics
}

func (n *NetworkState) GetPathMetrics(remote string, pathId uint64) *PathMetrics {
	if _, ok := n.Remotes[remote]; !ok {
		return nil
	}
	return n.Remotes[remote][pathId]
}
