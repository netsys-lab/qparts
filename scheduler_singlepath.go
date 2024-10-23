package qparts

type SchedulerSinglePath struct {
	index int
}

func NewSchedulerSinglePath() *SchedulerSinglePath {
	return &SchedulerSinglePath{
		index: 0,
	}
}

func (sr *SchedulerSinglePath) OnCongestionEvent(event *CongestionEvent) error {
	return nil
}

func (sr *SchedulerSinglePath) ScheduleWrite(data []byte, stream *PartsStream, dpStreams map[uint64]*QPartsDataplaneStream) SchedulingDecision {
	// Log.Info(stream.Conn.remote)
	/*s := stream.conn.remote.String()

	rem, ok := state.Remotes[s]
	if !ok {
		Log.Info("Adding remote")
		state.AddRemote(stream.conn.remote)
		rem = state.Remotes[s]
	}*/

	var s *QPartsDataplaneStream
	for _, p := range dpStreams {
		s = p
		break
	}

	da := DataAssignment{
		// Path: rem.Paths[sr.index],
		DataplaneStream: s, // dpStreams[0],
		Data:            data,
	}

	return SchedulingDecision{
		Assignments: []DataAssignment{da},
	}
}
