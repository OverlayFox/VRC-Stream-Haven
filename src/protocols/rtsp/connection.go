package rtsp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/bluenviron/gortsplib/v5"
	"github.com/bluenviron/gortsplib/v5/pkg/description"
	"github.com/bluenviron/gortsplib/v5/pkg/format"
	"github.com/bluenviron/gortsplib/v5/pkg/format/rtph264"
	"github.com/bluenviron/gortsplib/v5/pkg/format/rtpmpeg4audio"
	"github.com/bluenviron/mediacommon/v2/pkg/codecs/mpeg4audio"
	"github.com/pion/rtp"
	"github.com/rs/zerolog"
	"github.com/yapingcat/gomedia/go-codec"

	"github.com/OverlayFox/VRC-Stream-Haven/src/types"
)

type Connection struct {
	logger zerolog.Logger

	conn     net.Conn
	location types.Location

	stream        *gortsplib.ServerStream
	server        *gortsplib.Server
	session       *gortsplib.ServerSession
	h264Encoder   *rtph264.Encoder
	aacEncoder    *rtpmpeg4audio.Encoder
	aacSampleRate uint
	// onPlay        chan struct{}

	wg     sync.WaitGroup
	mtx    sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
}

func NewConnection(logger zerolog.Logger, upstreamCtx context.Context, conn net.Conn, location types.Location, server *gortsplib.Server, session *gortsplib.ServerSession) types.ConnectionRTSP {
	logger = logger.With().Str("ip", conn.RemoteAddr().String()).Str("location", location.String()).Logger()
	ctx, cancel := context.WithCancel(upstreamCtx)
	return &Connection{
		logger: logger,

		conn:     conn,
		location: location,

		server:  server,
		session: session,
		// onPlay:  make(chan struct{}, 1),

		ctx:    ctx,
		cancel: cancel,
	}
}

func (c *Connection) GetStream() *gortsplib.ServerStream {
	return c.stream
}

// StartPlay signals that the connection should start sending frames to the client.
func (c *Connection) StartPlay() error {
	if c.stream == nil {
		return errors.New("stream not initialized yet")
	}

	// select {
	// case <-c.ctx.Done():
	// 	return errors.New("connection context cancelled")
	// case c.onPlay <- struct{}{}:
	// default:
	// }

	return nil
}

func (c *Connection) GetAddr() net.Addr {
	return c.conn.RemoteAddr()
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

	err = c.primeEncoders(sps, pps, asc)
	if err != nil {
		return fmt.Errorf("failed to prime encoders: %w", err)
	}

	c.handleFrames(videoCh, 0, c.encodeH264)
	c.handleFrames(audioCh, 1, c.encodeAAC)

	return nil
}

func (c *Connection) Close() {
	c.cancel()
	err := c.conn.Close()
	if err != nil {
		c.logger.Error().Err(err).Msg("failed to close RTSP connection")
	}
	c.wg.Wait()

	c.logger.Info().Msg("RTSP connection closed")
}

// extractMetadata listens to the provided video and audio channels until it successfully extracts the SPS/PPS for video and ASC for audio.
//
// TODO: move this into the buffer for better optimisation
//
//nolint:gocognit // will be moved into the buffer and can be refactored then
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

// primeEncoders initializes the RTSP stream and encoders with the provided SPS, PPS, and ASC metadata and reels up the AAC and H264 encoder.
func (c *Connection) primeEncoders(sps, pps []byte, asc *mpeg4audio.AudioSpecificConfig) error {
	c.stream = &gortsplib.ServerStream{
		Server: c.server,
		Desc: &description.Session{
			Medias: []*description.Media{
				{
					Type: description.MediaTypeVideo,
					Formats: []format.Format{
						&format.H264{
							PayloadTyp:        96,
							PacketizationMode: 1,
							SPS:               sps,
							PPS:               pps,
						},
					},
				},
				{
					Type: description.MediaTypeAudio,
					Formats: []format.Format{&format.MPEG4Audio{
						PayloadTyp:       97,
						Config:           asc,
						SizeLength:       13,
						IndexLength:      3,
						IndexDeltaLength: 3,
					}},
				},
			},
		},
	}
	err := c.stream.Initialize()
	if err != nil {
		return fmt.Errorf("failed to initialize RTSP stream: %w", err)
	}

	// Create H264 Encoder and initialize it
	formatH264, ok := c.stream.Desc.Medias[0].Formats[0].(*format.H264)
	if !ok {
		return errors.New("failed to assert video format as H264")
	}
	h264Encoder, err := formatH264.CreateEncoder()
	if err != nil {
		return err
	}
	h264Encoder.PayloadMaxSize = 1450

	c.h264Encoder = h264Encoder
	err = c.h264Encoder.Init()
	if err != nil {
		return fmt.Errorf("failed to initialize H264 encoder: %w", err)
	}

	// Create AAC Encoder and initialize it
	formatAAC, ok := c.stream.Desc.Medias[1].Formats[0].(*format.MPEG4Audio)
	if !ok {
		return errors.New("failed to assert audio format as MPEG4Audio")
	}
	aacEncoder, err := formatAAC.CreateEncoder()
	if err != nil {
		return err
	}
	c.aacEncoder = aacEncoder

	if asc.SampleRate < 0 {
		return fmt.Errorf("invalid sample rate in ASC: %d", asc.SampleRate)
	}
	c.aacSampleRate = uint(asc.SampleRate)

	return c.aacEncoder.Init()
}

func (c *Connection) handleFrames(packetCh <-chan types.Frame, mediaIndex int, encoderFunc func(frame types.Frame) ([]*rtp.Packet, error)) {
	c.wg.Go(func() {
		// select {
		// case <-c.onPlay: // block until we receive the signal to start playing
		// 	c.logger.Debug().Msg("Received play signal, starting to server frames to client")
		// case <-c.ctx.Done():
		// 	return
		// }

		for {
			select {
			case <-c.ctx.Done():
				return
			case frame, ok := <-packetCh:
				if !ok {
					return
				}

				pkts, err := encoderFunc(frame)
				if err != nil {
					c.logger.Error().Err(err).Msg("failed to encode frame")
					frame.Decommission()
					return // TODO: handle this better, maybe try to re-prime the encoder if we fail to encode a frame?
				}

				c.mtx.RLock()
				media := c.stream.Desc.Medias[mediaIndex]
				c.mtx.RUnlock()

				for _, pkt := range pkts {
					if err = c.stream.WritePacketRTP(media, pkt); err != nil {
						c.logger.Error().Err(err).Msg("failed to write RTP packet to stream")
						continue
					}
				}
				frame.Decommission()
			}
		}
	})
}

func (c *Connection) encodeH264(frame types.Frame) ([]*rtp.Packet, error) {
	frameData := frame.Data()
	var nalus [][]byte
	codec.SplitFrame(frameData, func(nalu []byte) bool {
		naluCopy := make([]byte, len(nalu))
		copy(naluCopy, nalu)
		nalus = append(nalus, naluCopy)
		return true
	})

	if len(nalus) == 0 {
		return nil, nil
	}

	pkts, err := c.h264Encoder.Encode(nalus)
	if err != nil {
		return nil, fmt.Errorf("failed to encode H264 frame: %w", err)
	}

	baseNanos := frame.Header().Pts.Nanoseconds()
	if baseNanos < 0 {
		return nil, fmt.Errorf("invalid PTS for H264 frame: %d", baseNanos)
	}
	rtpTimestamp := uint32(uint64(baseNanos) * 9 / 100000) //nolint:gosec // timestamp wrapping is intentional behavior per the RTP specification (RFC 3550).
	for _, pkt := range pkts {
		pkt.Timestamp = rtpTimestamp
	}

	return pkts, nil
}

func (c *Connection) encodeAAC(frame types.Frame) ([]*rtp.Packet, error) {
	frameData := frame.Data()
	basePts := frame.Header().Pts

	// Split AAC frames and strip ADTS headers
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
		return nil, errors.New("no AAC frames found in audio frame")
	}

	pkts, err := c.aacEncoder.Encode(aacs)
	if err != nil {
		return nil, err
	}

	// Set timestamps for the generated packets
	baseNanos := basePts.Nanoseconds()
	if baseNanos < 0 {
		return nil, fmt.Errorf("invalid PTS for AAC frame: %d", baseNanos)
	}
	baseRtpTimestamp := uint32(uint64(baseNanos) * uint64(c.aacSampleRate) / 1000000000) //nolint:gosec // timestamp wrapping is intentional behavior per the RTP specification (RFC 3550).
	for _, pkt := range pkts {
		pkt.Timestamp = baseRtpTimestamp
	}

	return pkts, nil
}
