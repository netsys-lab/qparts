package qparts

import (
	"math/rand"
	"sync"
)

type QPartsSequenceCompletion struct {
	sync.Mutex
	StreamId       uint64
	SequenceId     uint64
	Parts          int
	SequenceSize   uint64
	CompletedParts int
	Data           []byte
	internalParts  [][]byte
	Completed      bool
}

func (sc *QPartsSequenceCompletion) AddPart(index int, data []byte) bool {
	sc.Lock()
	defer sc.Unlock()
	// qplogging.Log.Debug("Adding part ", index)
	sc.CompletedParts++
	sc.internalParts[index] = data

	// Copy into data starting from index
	// qplogging.Log.Debugf("Copying data %x from %d to %d \n", sha256.Sum256(data), index, index+len(data))
	//
	compl := sc.IsComplete()
	if compl {
		offset := 0
		for _, part := range sc.internalParts {
			copy(sc.Data[offset:], part)
			offset += len(part)
		}
		sc.Completed = true
		// qplogging.Log.Debugf("COMPLETE Hash %x\n", sha256.Sum256(sc.Data))
	}
	return compl
}

func (sc *QPartsSequenceCompletion) IsComplete() bool {
	return sc.Parts == sc.CompletedParts
}

type CompletionStore struct {
	sync.Mutex
	Completions map[uint64]*QPartsSequenceCompletion
}

func NewCompletionStore() *CompletionStore {
	return &CompletionStore{
		Completions: make(map[uint64]*QPartsSequenceCompletion),
	}
}

func (cs *CompletionStore) AddCompletion(c *QPartsSequenceCompletion) {
	cs.Lock()
	defer cs.Unlock()
	cs.Completions[c.StreamId] = c
}

func (cs *CompletionStore) GetCompletion(sequenceId uint64) *QPartsSequenceCompletion {
	return cs.Completions[sequenceId]
}

func (cs *CompletionStore) RemoveCompletion(sequenceId uint64) {
	cs.Lock()
	defer cs.Unlock()
	delete(cs.Completions, sequenceId)
}

// TODO: Move to numbering.go and use crypto safe sequences
func newSequenceId() uint64 {
	// Generate Random number
	return rand.Uint64()
}

func (cs *CompletionStore) GetOrCreateSequenceCompletion(streamId uint64, sequenceId uint64, numParts uint64, sequenceSize uint64) *QPartsSequenceCompletion {

	cs.Lock()

	defer cs.Unlock()

	if c, ok := cs.Completions[sequenceId]; ok {
		return c
	}

	pc := &QPartsSequenceCompletion{
		StreamId:      streamId,
		SequenceId:    sequenceId,
		Parts:         int(numParts),
		SequenceSize:  sequenceSize,
		internalParts: make([][]byte, numParts),
	}

	pc.Data = make([]byte, sequenceSize)
	cs.Completions[sequenceId] = pc

	return pc
}

func (cs *CompletionStore) NewSequenceCompletionFromSchedulingDecision(streamId uint64, decision *SchedulingDecision) *QPartsSequenceCompletion {

	pc := &QPartsSequenceCompletion{
		StreamId:   streamId,
		SequenceId: newSequenceId(),
		Parts:      len(decision.Assignments),
	}

	size := 0
	for _, da := range decision.Assignments {
		size += len(da.Data)
	}

	pc.Data = make([]byte, size)

	pc.SequenceSize = uint64(size)

	return pc
}
