package types

import (
	"time"

	"github.com/bluenviron/gortsplib/v4/pkg/description"
)

type FrameHeader struct {
	Type description.MediaType
	Pts  time.Duration // Time since the start of the stream
	Dts  time.Duration // Time since the start of the stream
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
