package types

import (
	"context"
	"time"

	"github.com/datarhei/gosrt/packet"
)

type Buffer interface {
	Write(frame Frame) error
	Subscribe(ctx context.Context, ptsOffsetToLive time.Duration) ([]BufferFlow, error)
	SubscribeInterleaved(ctx context.Context, ptsOffsetToLive time.Duration) ([]BufferFlow, error)
	IsReady() bool
	Close()
}

type BufferStream interface {
	IsReady() bool
	Write(frame Frame) error
	Subscribe(ctx context.Context, opts ...SubscribeOption) ([]BufferFlow, error)
	Cancel()
	Close()
}

type PacketBuffer interface {
	Subscribe() chan packet.Packet
	Unsubscribe(chan packet.Packet)
	Write(packet.Packet)
	Close()
}

// MediaType represents the type of a media stream.
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

type BufferFlow struct {
	Type       BufferType
	Channel    chan Frame
	Identifier string
	FirstPTS   time.Duration
}

type SubscribeConfig struct {
	PTSOffsetToLive *time.Duration
	DesiredPTSStart *time.Duration
}

type SubscribeOption func(*SubscribeConfig)

func SubscribeOptionWithPTSOffsetToLive(offset time.Duration) SubscribeOption {
	return func(cfg *SubscribeConfig) {
		cfg.PTSOffsetToLive = &offset
	}
}

func SubscribeOptionWithPTSOffsetToLiveAndWithPTSStart(start, offset time.Duration) SubscribeOption {
	return func(cfg *SubscribeConfig) {
		cfg.DesiredPTSStart = &start
		cfg.PTSOffsetToLive = &offset
	}
}
