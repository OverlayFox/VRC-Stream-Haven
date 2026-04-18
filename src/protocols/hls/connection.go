package hls

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/bluenviron/gohlslib/v2"
	"github.com/bluenviron/gohlslib/v2/pkg/codecs"
	"github.com/bluenviron/gortsplib/v5"
	"github.com/bluenviron/mediacommon/v2/pkg/codecs/mpeg4audio"
	"github.com/rs/zerolog"
	"github.com/yapingcat/gomedia/go-codec"

	"github.com/OverlayFox/VRC-Stream-Haven/src/types"
)

type Connection struct {
	logger zerolog.Logger

	location types.Location

	muxer         *gohlslib.Muxer
	videoTrack    *gohlslib.Track
	audioTrack    *gohlslib.Track
	aacSampleRate int

	wg     sync.WaitGroup
	mtx    sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
}

func NewConnection(upstreamCtx context.Context, logger zerolog.Logger, location types.Location) types.ConnectionRTSP {
	// logger = logger.With().Str("protocol", "hls").Str("location", location.String()).Logger()
	logger = logger.With().Str("protocol", "hls").Logger()
	ctx, cancel := context.WithCancel(upstreamCtx)
	return &Connection{
		logger:   logger,
		location: location,
		ctx:      ctx,
		cancel:   cancel,
	}
}

func (c *Connection) GetStream() *gortsplib.ServerStream {
	return nil
}

// HandleHTTP serves the underlying LL-HLS multiplexer to HTTP clients.
func (c *Connection) HandleHTTP(w http.ResponseWriter, r *http.Request) {
	c.mtx.RLock()
	muxer := c.muxer
	c.mtx.RUnlock()

	if muxer != nil {
		muxer.Handle(w, r)
	} else {
		http.Error(w, "Stream not primed yet", http.StatusServiceUnavailable)
	}
}

func (c *Connection) StartPlay() error {
	return nil
}

func (c *Connection) GetAddr() net.Addr {
	// Dummy address since this multiplexer serves many HTTP clients.
	return &net.TCPAddr{IP: net.ParseIP("0.0.0.0"), Port: 0}
}

func (c *Connection) GetType() types.ConnectionType {
	return types.ConnectionTypeReader
}

func (c *Connection) GetCtx() context.Context {
	return c.ctx
}

func (c *Connection) GetLocation() types.Location {
	return c.location
}

func (c *Connection) GetLogger() zerolog.Logger {
	return c.logger
}

func (c *Connection) Write(streams []types.BufferOutput) error {
	var audioCh, videoCh chan types.Frame
	for _, stream := range streams {
		switch stream.Type {
		case types.BufferTypeVideo:
			videoCh = stream.Channel
		case types.BufferTypeAudio:
			audioCh = stream.Channel
		}
	}
	if audioCh == nil || videoCh == nil {
		return errors.New("missing audio or video stream from haven")
	}

	sps, pps, asc, err := c.extractMetadata(&videoCh, &audioCh)
	if err != nil {
		return fmt.Errorf("failed to extract streams metadata: %w", err)
	}

	err = c.primeMuxer(sps, pps, asc)
	if err != nil {
		return fmt.Errorf("failed to prime HLS muxer: %w", err)
	}

	c.handleFrames(videoCh, c.writeH264)
	c.handleFrames(audioCh, c.writeAAC)

	return nil
}

func (c *Connection) Close() {
	c.cancel()
	c.wg.Wait()

	c.mtx.Lock()
	if c.muxer != nil {
		c.muxer.Close()
	}
	c.mtx.Unlock()

	c.logger.Info().Msg("HLS connection closed")
}

//
// Helper functions
//

// extractMetadata reads from the provided video and audio channels and writes them to new channels in place.
// It extracts the SPS/PPS from the video stream and the Audio Specific Config from the audio stream, which are needed to prime the HLS muxer.
//
//nolint:gocognit // This function is a bit complex due to the need to read from both channels concurrently and wait until both metadata are extracted before returning.
func (c *Connection) extractMetadata(videoCh, audioCh *chan types.Frame) (sps, pps []byte, asc *mpeg4audio.AudioSpecificConfig, err error) {
	upstreamVideoCh := *videoCh
	upstreamAudioCh := *audioCh
	*videoCh = make(chan types.Frame, cap(upstreamVideoCh))
	*audioCh = make(chan types.Frame, cap(upstreamAudioCh))

	receiveDone := make(chan struct{}, 2)

	c.wg.Go(func() {
		defer close(receiveDone)
		defer close(*videoCh)
		defer close(*audioCh)

		for {
			select {
			case frame, ok := <-upstreamVideoCh:
				if !ok {
					return
				}
				if sps != nil && pps != nil {
					*videoCh <- frame
					continue
				}

				if sps == nil || pps == nil {
					extractedSps, extractedPps, err := ExtractSPSPPS(frame)
					if err != nil {
						c.logger.Debug().Err(err).Msg("SPS/PPS not in this frame, continuing")
					} else {
						sps, pps = extractedSps, extractedPps
						if sps != nil && pps != nil && asc != nil {
							receiveDone <- struct{}{}
						}
					}
				}
				*videoCh <- frame

			case frame, ok := <-upstreamAudioCh:
				if !ok {
					return
				}
				if asc != nil {
					*audioCh <- frame
					continue
				}

				if asc == nil {
					extractedAsc, err := ExtractASC(frame)
					if err != nil {
						c.logger.Debug().Err(err).Msg("ASC not in this frame, continuing")
					} else {
						asc = extractedAsc
						if sps != nil && pps != nil && asc != nil {
							receiveDone <- struct{}{}
						}
					}
				}
				*audioCh <- frame
			}
		}
	})

	select {
	case <-receiveDone:
		return sps, pps, asc, nil
	case <-c.ctx.Done():
		return nil, nil, nil, errors.New("context cancelled while waiting for metadata")
	}
}

func (c *Connection) primeMuxer(sps, pps []byte, asc *mpeg4audio.AudioSpecificConfig) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	c.videoTrack = &gohlslib.Track{
		Codec: &codecs.H264{
			SPS: sps,
			PPS: pps,
		},
		ClockRate: 90000,
	}

	c.audioTrack = &gohlslib.Track{
		Codec: &codecs.MPEG4Audio{
			Config: *asc,
		},
		ClockRate: asc.SampleRate,
	}
	c.aacSampleRate = asc.SampleRate

	c.muxer = &gohlslib.Muxer{
		Variant:            gohlslib.MuxerVariantMPEGTS,
		SegmentCount:       5,
		SegmentMinDuration: 1 * time.Second,
		Tracks:             []*gohlslib.Track{c.videoTrack, c.audioTrack},
		// Explicitly leaving Directory blank guarantees segments remain purely in RAM
	}

	return c.muxer.Start()
}

func (c *Connection) handleFrames(packetCh <-chan types.Frame, muxFunc func(frame types.Frame) error) {
	c.wg.Go(func() {
		for {
			select {
			case <-c.ctx.Done():
				return
			case frame, ok := <-packetCh:
				if !ok {
					return
				}

				if err := muxFunc(frame); err != nil {
					c.logger.Error().Err(err).Msg("failed to mux frame")
				}
				frame.Decommission()
			}
		}
	})
}

func (c *Connection) writeH264(frame types.Frame) error {
	frameData := frame.Data()
	var nalus [][]byte

	codec.SplitFrame(frameData, func(nalu []byte) bool {
		naluCopy := make([]byte, len(nalu))
		copy(naluCopy, nalu)
		nalus = append(nalus, naluCopy)
		return true
	})

	if len(nalus) == 0 {
		return nil
	}

	baseNanos := frame.Header().Pts.Nanoseconds()
	if baseNanos < 0 {
		return fmt.Errorf("invalid PTS for H264 frame: %d", baseNanos)
	}

	pts := (baseNanos * 90000) / 1000000000
	c.mtx.Lock()
	defer c.mtx.Unlock()

	return c.muxer.WriteH264(c.videoTrack, time.Now(), pts, nalus)
}

func (c *Connection) writeAAC(frame types.Frame) error {
	frameData := frame.Data()
	basePts := frame.Header().Pts

	var aacs [][]byte
	codec.SplitAACFrame(frameData, func(aac []byte) {
		var adts codec.ADTS_Frame_Header
		adts.Decode(aac)

		headerLen := 7
		if adts.Fix_Header.Protection_absent == 0 {
			headerLen = 9
		}

		payload := make([]byte, len(aac)-headerLen)
		copy(payload, aac[headerLen:])
		aacs = append(aacs, payload)
	})
	if len(aacs) == 0 {
		return errors.New("no AAC frames found in audio frame")
	}

	baseNanos := basePts.Nanoseconds()
	if baseNanos < 0 {
		return fmt.Errorf("invalid PTS for AAC frame: %d", baseNanos)
	}

	pts := (baseNanos * int64(c.aacSampleRate)) / 1000000000
	c.mtx.Lock()
	defer c.mtx.Unlock()

	return c.muxer.WriteMPEG4Audio(c.audioTrack, time.Now(), pts, aacs)
}
