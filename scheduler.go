package qparts

import (
	"github.com/netsys-lab/qparts/pkg/qpnet"
	"github.com/netsys-lab/qparts/pkg/qpscion"
	"github.com/scionproto/scion/pkg/snet"
)

type IScheduler interface {
	ScheduleWrite(data []byte, stream *PartsStream, dpStreams map[uint64]*QPartsDataplaneStream) SchedulingDecision
	InitialPathSelection(preference uint32, paths []qpscion.QPartsPath) ([]qpscion.QPartsPath, error)
	OnCongestionEvent(event *qpnet.CongestionEvent) error
}

type Scheduler struct {
	plugins      []SchedulerPlugin
	activePlugin SchedulerPlugin
}

// We need to have the internal stream id here
type DataAssignment struct {
	// Path   *PartsPath
	DataplaneStream *QPartsDataplaneStream
	Data            []byte
	Remote          *snet.UDPAddr
}

type SchedulingDecision struct {
	Assignments []DataAssignment
}

type SchedulerPlugin interface {
	ScheduleWrite(data []byte, stream *PartsStream, dpStreams map[uint64]*QPartsDataplaneStream) SchedulingDecision
	InitialPathSelection(preference uint32, paths []qpscion.QPartsPath) ([]qpscion.QPartsPath, error)
	OnCongestionEvent(event *qpnet.CongestionEvent) error
}

func NewScheduler() *Scheduler {
	s := &Scheduler{
		plugins: make([]SchedulerPlugin, 0),
	}

	// s.ActivatePlugin(NewSchedulerRoundRobin())
	s.ActivatePlugin(NewSchedulerECMP())

	return s
}

func (s *Scheduler) OnCongestionEvent(event *qpnet.CongestionEvent) error {
	return s.activePlugin.OnCongestionEvent(event)
}

func (s *Scheduler) ActivatePlugin(p SchedulerPlugin) {
	// Ensure plugin only added once
	s.plugins = append(s.plugins, p)
	s.activePlugin = p
}

func (s *Scheduler) ScheduleWrite(data []byte, stream *PartsStream, dpStreams map[uint64]*QPartsDataplaneStream) SchedulingDecision {
	// TODO: State
	//Log.Info("Scheduling write, active plugin")
	//Log.Info(s.activePlugin)
	return s.activePlugin.ScheduleWrite(data, stream, dpStreams)
}

func (s *Scheduler) InitialPathSelection(preference uint32, paths []qpscion.QPartsPath) ([]qpscion.QPartsPath, error) {
	// Implement the logic for initial path selection here
	return s.activePlugin.InitialPathSelection(preference, paths)
}
