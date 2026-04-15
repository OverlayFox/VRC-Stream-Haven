package types

import (
	"time"

	"github.com/yapingcat/gomedia/go-codec"
)

type FrameHeader struct {
	Cid codec.CodecID
	Pts time.Duration // Time since the start of the stream
	Dts time.Duration // Time since the start of the stream
}

type Frame interface {
	// Decommission frees the payload. The frame shouldn't be uses afterwards.
	Decommission()
	// Clone clones a frame.
	Clone() Frame
	// Header returns a pointer to the frame header.
	Header() *FrameHeader
	// SetData replaces the payload of the frame with the provided one.
	SetData([]byte)
	// Data returns the payload the frame holds. The frame stays the
	// owner of the data, i.e. modifying the returned data will also
	// modify the payload.
	Data() []byte
	// Len return the length of the payload in the packet.
	Len() uint64
	// IsKeyFrame returns if the frame is a keyframe or not
	IsKeyFrame() bool
}

type FrameFormat int

const (
	FrameFormatUnknown FrameFormat = iota
	FrameFormatAnnexB
	FrameFormatAVCC
)

func (f FrameFormat) String() string {
	switch f {
	case FrameFormatAnnexB:
		return "FrameFormatAnnexB"
	case FrameFormatAVCC:
		return "FrameFormatAVCC"
	default:
		return "FrameFormatUnknown"
	}
}

// FrameHeap implements heap.Interface for types.Frame, ordered by DTS.
//
//nolint:recvcheck // Certain functions need to be pointers to make this to work
type FrameHeap []Frame

// Len returns the number of elements in the heap.
func (h FrameHeap) Len() int { return len(h) }

// Less compares two frames based on their DTS to ensure min-heap property (smallest DTS at root).
func (h FrameHeap) Less(i, j int) bool {
	if h[i].Header().Dts < h[j].Header().Dts {
		return true
	} else if h[i].Header().Dts > h[j].Header().Dts {
		return false
	}

	// If DTS are equal, prioritize video frames over audio frames
	isIVideo := h[i].Header().Cid == codec.CODECID_VIDEO_H264
	isJVideo := h[j].Header().Cid == codec.CODECID_VIDEO_H264

	// If both frames are of the same type, maintain their order based on PTS
	if (isIVideo && isJVideo) || (!isIVideo && !isJVideo) {
		// If PTS is also equal, we treat them as equivalent by returning false.
		// This is an edge case, as two distinct frames should not have identical DTS and PTS.
		return h[i].Header().Pts < h[j].Header().Pts
	}

	// Prioritize video frames over audio frames
	return isIVideo && !isJVideo
}

// Swap swaps two elements in the heap.
func (h FrameHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

// Push adds an element to the heap.
func (h *FrameHeap) Push(x any) {
	*h = append(*h, x.(Frame)) //nolint:errcheck // We ensure that only Frame types are pushed, so this should never panic
}

// Pop removes and returns the minimum element (root) from the heap.
func (h *FrameHeap) Pop() any {
	old := *h
	n := len(old)
	frame := old[n-1]
	*h = old[0 : n-1]
	return frame
}
