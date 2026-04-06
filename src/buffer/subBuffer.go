package buffer

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/OverlayFox/VRC-Stream-Haven/src/types"
	"github.com/rs/zerolog"
	"github.com/yapingcat/gomedia/go-codec"
)

const (
	MinKeyFrames = 3
	WaitForIDR   = 8 * time.Second
	BufCap       = 500
)

type subBuffer struct {
	logger zerolog.Logger

	circBuf *circularBuffer
	cap     int

	keyFrames *types.OrderedMap[time.Duration, int]
	curPTS    time.Duration
	startPTS  time.Duration
	started   bool

	format  types.FrameFormat
	bufType types.BufferType

	idrOnce    sync.Once
	idrExpired atomic.Bool

	mtx    sync.RWMutex
	ready  atomic.Bool
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

func newBufferStream(logger zerolog.Logger, bufType types.BufferType) types.SubBuffer {
	ctx, cancel := context.WithCancel(context.Background())
	return &subBuffer{
		logger:    logger,
		circBuf:   newCircularBuffer(logger.With().Str("process_name", fmt.Sprintf("circular_%s_buffer", bufType)).Logger(), BufCap),
		cap:       BufCap,
		keyFrames: types.NewOrderedMap[time.Duration, int](),
		format:    types.FrameFormatUnknown,
		bufType:   bufType,
		ctx:       ctx,
		cancel:    cancel,
	}
}

func (b *subBuffer) IsReady() bool {
	return b.ready.Load()
}

func (b *subBuffer) Write(frame types.Frame) error {
	switch b.bufType {
	case types.BufferTypeVideo:
		return b.writeVideo(frame)
	case types.BufferTypeAudio:
		return b.writeAudio(frame)
	default:
		frame.Decommission()
		return fmt.Errorf("unsupported buffer type: %s", b.bufType)
	}
}

func (b *subBuffer) writeVideo(frame types.Frame) error {
	if frame.Header().Cid != codec.CODECID_VIDEO_H264 {
		frame.Decommission()
		return fmt.Errorf("invalid video codec: %s", codec.CodecString(frame.Header().Cid))
	}
	b.idrOnce.Do(func() {
		b.wg.Go(func() {
			timer := time.NewTimer(WaitForIDR)
			defer timer.Stop()
			select {
			case <-b.ctx.Done():
				return
			case <-timer.C:
				b.idrExpired.Store(true)
			}
		})
	})
	var idr []byte
	var nalus [][]byte
	codec.SplitFrameWithStartCode(frame.Data(), func(nalu []byte) bool {
		nalus = append(nalus, nalu)
		return true
	})
	for _, nalu := range nalus {
		if codec.H264NaluType(nalu) == codec.H264_NAL_I_SLICE && codec.IsH264IDRFrame(nalu) {
			idr = nalu
			break
		}
	}

	if b.format == types.FrameFormatUnknown {
		if len(idr) == 0 {
			frame.Decommission()
			if b.idrExpired.Load() {
				return fmt.Errorf("no IDR frame in first %f seconds", WaitForIDR.Seconds())
			}
			return nil
		}
		ff, err := DetectH264Format(idr)
		if err != nil {
			frame.Decommission()
			return fmt.Errorf("detect H.264 format failed: %w", err)
		}
		b.format = ff
		b.logger.Debug().Msgf("detected H.264 format: %s", b.format)
	}

	pos := b.circBuf.Write(frame)
	b.mtx.Lock()
	b.curPTS = frame.Header().Pts
	b.keyFrames.Drop(frame.Header().Pts)
	if len(idr) > 0 {
		b.keyFrames.Set(frame.Header().Pts, pos)
	}
	if !b.started {
		b.startPTS = frame.Header().Pts
		b.started = true
	}
	b.mtx.Unlock()
	if !b.IsReady() && frame.Header().Cid == codec.CODECID_VIDEO_H264 {
		b.mtx.RLock()
		healthy := frame.Header().Pts-b.startPTS > 2*time.Second
		enough := b.keyFrames.Len() >= MinKeyFrames
		b.mtx.RUnlock()
		if healthy && enough {
			b.ready.Store(true)
			b.logger.Debug().Msg("video buffer ready")
		}
	}
	frame.Decommission()
	return nil
}

func (b *subBuffer) writeAudio(frame types.Frame) error {
	if frame.Header().Cid != codec.CODECID_AUDIO_AAC && frame.Header().Cid != codec.CODECID_AUDIO_MP3 {
		frame.Decommission()
		return fmt.Errorf("invalid audio codec: %s", codec.CodecString(frame.Header().Cid))
	}
	b.mtx.Lock()
	if !b.started {
		b.startPTS = frame.Header().Pts
		b.started = true
	}
	b.curPTS = frame.Header().Pts
	b.mtx.Unlock()
	b.circBuf.Write(frame)
	if b.bufType == types.BufferTypeAudio && !b.IsReady() {
		b.mtx.RLock()
		healthy := frame.Header().Pts-b.startPTS > 2*time.Second
		b.mtx.RUnlock()
		if healthy {
			b.ready.Store(true)
			b.logger.Debug().Msg("audio buffer ready")
		}
	}
	frame.Decommission()
	return nil
}

func (b *subBuffer) Subscribe(upstreamCtx context.Context, opts *types.SubscribeBuilder) ([]types.BufferOutput, error) {
	if !b.IsReady() {
		return nil, types.ErrBufferNotReady
	}

	var ch chan types.Frame
	var desiredPTS time.Duration

	// Get position to read from based on subscribe options
	if opts.PTSOffsetToLive != nil {
		startPos, closestPTS, err := b.getStartPos(*opts.PTSOffsetToLive, nil)
		if err != nil {
			return nil, err
		}
		ch, err = b.circBuf.ReadFromPos(startPos, upstreamCtx)
		if err != nil {
			return nil, err
		}
		desiredPTS = closestPTS
	}

	if opts.DesiredPTSStart != nil {
		startPos, closestPTS, err := b.getStartPos(0, opts.DesiredPTSStart)
		if err != nil {
			return nil, err
		}
		ch, err = b.circBuf.ReadFromPos(startPos, upstreamCtx)
		if err != nil {
			return nil, err
		}
		desiredPTS = closestPTS
	}

	var firstPTS time.Duration
	outCh := make(chan types.Frame, b.cap)
	b.wg.Go(func() {
		defer CloseAndDrain(outCh)
		first, ok := <-ch
		if !ok {
			return
		}

		preBuf := []types.Frame{first}
		firstPTS = first.Header().Pts
		lastPTS := firstPTS
		preDur := desiredPTS - firstPTS
		for lastPTS-firstPTS < preDur {
			select {
			case <-b.ctx.Done():
				return
			case <-upstreamCtx.Done():
				return
			case frame, ok := <-ch:
				if !ok {
					break
				}
				preBuf = append(preBuf, frame)
				lastPTS = frame.Header().Pts
			}
		}
		for _, frame := range preBuf {
			select {
			case <-b.ctx.Done():
				return
			case <-upstreamCtx.Done():
				return
			case outCh <- frame:
			default:
				b.logger.Warn().Msg("downstream full, skipping prebuffer frame")
				frame.Decommission()
			}
		}
		for {
			select {
			case <-b.ctx.Done():
				return
			case <-upstreamCtx.Done():
				return
			case frame, ok := <-ch:
				if !ok {
					return
				}
				select {
				case outCh <- frame:
				default:
					b.logger.Warn().Msg("downstream full, skipping live frame")
					frame.Decommission()
				}
			}
		}
	})
	return []types.BufferOutput{{
		Channel:  outCh,
		Type:     b.bufType,
		Title:    "PGM",
		StartPTS: firstPTS,
	}}, nil
}

func (b *subBuffer) Cancel() {
	b.cancel()
}

func (b *subBuffer) Close() {
	b.logger.Debug().Msgf("closing buffer stream '%s'", b.bufType.String())
	b.wg.Wait()
	b.mtx.Lock()
	defer b.mtx.Unlock()
	b.ready.Store(false)
	if b.circBuf != nil {
		b.circBuf.Close()
		b.circBuf = nil
	}
	b.started = false
	b.startPTS = 0
	b.curPTS = 0
	b.keyFrames = nil
	b.format = types.FrameFormatUnknown
}

func (b *subBuffer) getStartPos(offset time.Duration, desiredPTS *time.Duration) (pos int, closestPTS time.Duration, err error) {
	if !b.IsReady() {
		return 0, 0, types.ErrBufferNotReady
	}
	b.mtx.RLock()
	defer b.mtx.RUnlock()
	if desiredPTS != nil {
		return b.getStartPosFromPTS(*desiredPTS, b.circBuf.writePos-1)
	}
	if b.bufType == types.BufferTypeAudio {
		target := b.curPTS - offset
		newest := (b.circBuf.writePos - 1 + b.cap) % b.cap
		return b.getStartPosFromPTS(target, newest)
	}
	if b.keyFrames.Len() == 0 {
		return 0, 0, errors.New("no keyframes in buffer")
	}
	target := b.curPTS - offset
	keys := b.keyFrames.Keys()
	newestIdx := len(keys) - 1
	closestPTS = keys[newestIdx]
	minDiff := (target - closestPTS).Abs()
	for i := newestIdx - 1; i >= 0; i-- {
		k := keys[i]
		diff := (target - k).Abs()
		if diff <= minDiff {
			minDiff = diff
			closestPTS = k
		} else {
			break
		}
	}
	pos, ok := b.keyFrames.Get(closestPTS)
	if !ok {
		return 0, 0, fmt.Errorf("no keyframe for pts %s", closestPTS)
	}
	return pos, closestPTS, nil
}

func (b *subBuffer) getStartPosFromPTS(targetPTS time.Duration, newestPos int) (int, time.Duration, error) {
	peeked, err := b.circBuf.peek(newestPos)
	if err != nil {
		return 0, 0, err
	}
	minDiff := (targetPTS - peeked.Header().Pts).Abs()
	closestPos := newestPos
	closestPTS := peeked.Header().Pts
	for i := 1; i < b.cap; i++ {
		pos := (newestPos - i + b.cap) % b.cap
		peekedFrame, err := b.circBuf.peek(pos)
		if err != nil {
			return closestPos, closestPTS, err
		}
		diff := (targetPTS - peekedFrame.Header().Pts).Abs()
		if diff <= minDiff {
			minDiff = diff
			closestPos = pos
			closestPTS = peekedFrame.Header().Pts
		} else {
			break
		}
	}
	return closestPos, closestPTS, nil
}
