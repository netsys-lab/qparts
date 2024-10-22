/*
 * In this file we implement the state of a part of a stream that will be sent or received over a path
 * PartCompletion is a struct that represents a completed part of a stream.
 * It contains the part ID, stream ID, sequence number, size, data, and a list
 * of missing sequence numbers.
 * The list may be optimized later into some bitmap or other data structure.
 */

package qparts

type PartCompletion struct {
	PartId     uint64
	StreamId   uint64
	SequenceNo uint64
	Size       int
	Len        int
	PathId     uint32
	Data       []byte
	MissingNos []uint64
	Round      uint32
	index      int
}

func (pc *PartCompletion) NextSequenceNo() uint64 {
	pc.SequenceNo++
	return pc.SequenceNo
}

func (pc *PartCompletion) AddSequenceNo(sequenceNo uint64, data []byte) {
	// TODO: Boundary Check
	copy(pc.Data[pc.index:len(data)], data)
	pc.index += len(data)
	pc.Len += 1
}

func (pc *PartCompletion) Done() bool {
	return pc.Len == pc.Size
}
