package qparts

import "fmt"

type SchedulerRoundRobin struct {
	index int
}

func NewSchedulerRoundRobin() *SchedulerRoundRobin {
	return &SchedulerRoundRobin{
		index: 0,
	}
}

func (sr *SchedulerRoundRobin) ScheduleWrite(data []byte, stream *PartsStream, state *NetworkState) SchedulingDecision {

	s := stream.conn.remote.String()
	Log.Info("Scheduling write for index", sr.index)
	da := DataAssignment{
		Path: state.Remotes[s].Paths[sr.index],
		Data: data,
	}

	Log.Info("Got paths available ", state.Remotes[s].Paths)

	for i, p := range state.Remotes[s].Paths {
		fmt.Printf("Path %d has id %d\n", i, p.Id)
	}

	sr.index++

	if sr.index >= len(state.Remotes[s].Paths) {
		Log.Info("Resetting index")
		sr.index = 0
	}

	return SchedulingDecision{
		Assignments: []DataAssignment{da},
	}
}
