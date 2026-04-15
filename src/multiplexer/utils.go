package multiplexer

import (
	"github.com/asticode/go-astits"
	"github.com/yapingcat/gomedia/go-codec"
)

func mapStreamType(st astits.StreamType) codec.CodecID {
	switch st {
	case astits.StreamTypeAACAudio:
		return codec.CODECID_AUDIO_AAC
	case astits.StreamTypeH264Video:
		return codec.CODECID_VIDEO_H264
	default:
		return codec.CODECID_UNRECOGNIZED
	}
}

func unwrapTs(cur, last int64, set bool) int64 {
	if !set {
		return cur
	}
	diff := cur - (last % maxTsClock)
	if diff > maxTsClock/2 {
		return last + diff - maxTsClock
	} else if diff < -maxTsClock/2 {
		return last + diff + maxTsClock
	}
	return last + diff
}
