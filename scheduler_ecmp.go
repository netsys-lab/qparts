package qparts

/*
type SchedulerECMP struct {
	index int
}

func NewSchedulerECMP() *SchedulerECMP {
	return &SchedulerECMP{
		index: 0,
	}
}

func (sr *SchedulerECMP) OnCongestionEvent(event *CongestionEvent) error {
	return nil
}

func (sr *SchedulerECMP) ScheduleWrite(data []byte, stream *PartsStream, state *NetworkState) SchedulingDecision {
	// Log.Info(stream.Conn.remote)
	s := stream.conn.remote.String()

	rem, ok := state.Remotes[s]
	if !ok {
		Log.Info("Adding remote")
		state.AddRemote(stream.conn.remote)
		rem = state.Remotes[s]
	}

	da := DataAssignment{
		Path: rem.Paths[sr.index],
		Data: data,
	}

	return SchedulingDecision{
		Assignments: []DataAssignment{da},
	}
}
*/
