package qpmetrics

import (
	"sync"
	"time"
)

type PathMetrics struct {
	RTT                uint64
	Packetloss         uint64
	Throughput         uint64
	PathId             uint64
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

func (n *NetworkState) StartMeasurements() {
	go n.runDetectionTicker()
}

func (n *NetworkState) runDetectionTicker() {
	ticker := time.NewTicker(250 * time.Millisecond)
	for range ticker.C {
		n.DetectCongestion()
	}
}

func (n *NetworkState) DetectCongestion() {
	for _, rem := range n.Remotes {
		for _, path := range rem {
			if path.Packetloss > 0 {
				n.OnCongestionEvent[0](&CongestionEvent{
					Reason: "Packetloss",
					CongestionPaths: []PathCongestion{
						{
							// PathId:         path.PathId,
							ThroughputDrop: 0,
							PacketLoss:     float64(path.Packetloss),
						},
					},
				})
			}
		}
	}
}

func (n *NetworkState) AddCongestionEventHandler(handler func(event *CongestionEvent) error) {
	n.OnCongestionEvent = append(n.OnCongestionEvent, handler)
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
