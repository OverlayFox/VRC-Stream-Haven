package types

import (
	"context"
	"time"
)

type Buffer interface {
	Write(frame Frame) error
	Subscribe(ctx context.Context, ptsOffsetToLive time.Duration) ([]BufferOutput, error)
	IsReady() bool
	Close()
}

type SubBuffer interface {
	IsReady() bool
	Write(frame Frame) error
	SubscribePTS(ctx context.Context, desiredPTS time.Duration) ([]BufferOutput, error)
	SubscribeOffset(ctx context.Context, offsetToLive time.Duration) ([]BufferOutput, error)
	Cancel()
	Close()
}

// BufferType represents the type of a media stream.
type BufferType int

const (
	BufferTypeUnkown BufferType = iota
	BufferTypeVideo
	BufferTypeAudio
	BufferTypeInterleaved
)

var mediaTypeToString = map[BufferType]string{
	BufferTypeUnkown:      "unknown",
	BufferTypeVideo:       "video",
	BufferTypeAudio:       "audio",
	BufferTypeInterleaved: "interleaved",
}

var stringToMediaType = make(map[string]BufferType)

//nolint:gochecknoinits // init function to populate the reverse map for BufferType string representations.
func init() {
	// This loop runs once, creating the reverse map automatically.
	// This ensures the two maps are always in sync.
	for mt, s := range mediaTypeToString {
		stringToMediaType[s] = mt
	}
}

func (mt BufferType) String() string {
	if s, ok := mediaTypeToString[mt]; ok {
		return s
	}
	return mediaTypeToString[BufferTypeUnkown]
}

func MediaTypeFromString(s string) BufferType {
	if mt, ok := stringToMediaType[s]; ok {
		return mt
	}
	return BufferTypeUnkown
}

type BufferOutput struct {
	Title    string
	Type     BufferType
	Channel  chan Frame
	StartPTS time.Duration // StartPTS is the PTS of the first frame in the channel, used for synchronization.
}

type SubscribeBuilder struct {
	PTSOffsetToLive *time.Duration
	DesiredPTSStart *time.Duration
}

func NewSubscribeBuilder() *SubscribeBuilder {
	return &SubscribeBuilder{}
}

func (s *SubscribeBuilder) WithPTSOffsetToLive(offset time.Duration) *SubscribeBuilder {
	s.DesiredPTSStart = nil
	s.PTSOffsetToLive = &offset
	return s
}

func (s *SubscribeBuilder) WithPTSStart(start time.Duration) *SubscribeBuilder {
	s.DesiredPTSStart = &start
	s.PTSOffsetToLive = nil
	return s
}
