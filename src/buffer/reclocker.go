package buffer

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/OverlayFox/VRC-Stream-Haven/src/types"
	"github.com/yapingcat/gomedia/go-codec"
)

var (
	ErrBadTimestamp = errors.New("invalid frame timestamp")
	ErrDtsBackwards = errors.New("dts cannot go backwards")
	ErrNegativeTs   = errors.New("timestamp negative")
)

type streamTs struct {
	curDts time.Duration
	set    bool
}

type reclocker struct {
	offset     time.Duration
	firstFrame bool
	streams    map[codec.CodecID]*streamTs
	mu         sync.Mutex
}

func newReclocker() *reclocker {
	return &reclocker{
		streams: make(map[codec.CodecID]*streamTs),
	}
}

func (r *reclocker) AddStream(cid codec.CodecID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.streams[cid]; !ok {
		r.streams[cid] = &streamTs{}
	}
}

func (r *reclocker) Reclock(f types.Frame) error {
	if f.Header().Dts > f.Header().Pts {
		return fmt.Errorf("%w: dts '%d' > pts '%d'", ErrBadTimestamp, f.Header().Dts.Milliseconds(), f.Header().Pts.Milliseconds())
	}

	if !r.firstFrame {
		r.offset = f.Header().Dts
		f.Header().Dts = 0
		f.Header().Pts = f.Header().Pts - r.offset
		r.firstFrame = true

		return nil
	}

	f.Header().Pts -= r.offset
	f.Header().Dts -= r.offset

	if f.Header().Dts < 0 || f.Header().Pts < 0 {
		return fmt.Errorf("%w: dts '%d', pts '%d'", ErrNegativeTs, f.Header().Dts.Milliseconds(), f.Header().Pts.Milliseconds())
	}

	stream, ok := r.streams[f.Header().Cid]
	if !ok {
		return fmt.Errorf("stream id %s not registered", codec.CodecString(f.Header().Cid))
	}

	if !stream.set {
		stream.curDts = f.Header().Dts
		stream.set = true
	}

	if f.Header().Dts < stream.curDts {
		return fmt.Errorf("%w: frame dts '%d' < cur dts '%d'", ErrDtsBackwards, f.Header().Dts.Milliseconds(), stream.curDts.Milliseconds())
	}
	stream.curDts = f.Header().Dts

	return nil
}

func (r *reclocker) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.offset = 0
	r.firstFrame = false
	for _, s := range r.streams {
		s.set = false
		s.curDts = 0
	}
	r.streams = make(map[codec.CodecID]*streamTs)
}
