package rtsp

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/OverlayFox/VRC-Stream-Haven/src/types"
	"github.com/bluenviron/gortsplib/v5"
	"github.com/bluenviron/gortsplib/v5/pkg/base"
	"github.com/bluenviron/gortsplib/v5/pkg/description"
	"github.com/bluenviron/gortsplib/v5/pkg/format"
	"github.com/bluenviron/gortsplib/v5/pkg/format/rtph264"
	"github.com/bluenviron/gortsplib/v5/pkg/format/rtpmpeg4audio"
	"github.com/bluenviron/mediacommon/v2/pkg/codecs/mpeg4audio"
	"github.com/pion/rtp"
	"github.com/rs/zerolog"
	"github.com/yapingcat/gomedia/go-codec"
)

type connectionHandler struct {
	logger zerolog.Logger

	server *gortsplib.Server
	stream *gortsplib.ServerStream

	aacEncoder  *rtpmpeg4audio.Encoder
	h264Encoder *rtph264.Encoder

	haven    types.Haven
	locator  types.Locator
	location types.Location

	isFlagship bool
	rtpEncoder *rtph264.Encoder

	mu sync.RWMutex
	wg sync.WaitGroup
}

// OnConnOpen is called when a connection is opened.
func (sh *connectionHandler) OnConnOpen(ctx *gortsplib.ServerHandlerOnConnOpenCtx) {
	location, err := sh.locator.GetLocation(ctx.Conn.NetConn().RemoteAddr())
	if err != nil {
		sh.logger.Warn().Err(err).Msg("failed to get location")
		sh.location = types.Location{}
	} else {
		sh.location = location
	}
	sh.logger.Debug().Str("client_ip", ctx.Conn.NetConn().RemoteAddr().String()).Msg("rtsp connection opened")
}

// OnConnClose is called when a connection is closed.
func (sh *connectionHandler) OnConnClose(ctx *gortsplib.ServerHandlerOnConnCloseCtx) {
	sh.logger.Debug().Str("client_ip", ctx.Conn.NetConn().RemoteAddr().String()).Msgf("rtsp connection closed: %v", ctx.Error)
}

// OnSessionOpen is called when a session is opened.
func (sh *connectionHandler) OnSessionOpen(ctx *gortsplib.ServerHandlerOnSessionOpenCtx) {
	sh.logger.Debug().Str("client_ip", ctx.Conn.NetConn().RemoteAddr().String()).Msg("rtsp session opened")
}

// OnSessionClose is called when a session is closed.
func (sh *connectionHandler) OnSessionClose(_ *gortsplib.ServerHandlerOnSessionCloseCtx) {
	sh.logger.Debug().Msg("rtsp session closed")
}

// OnDescribe is called when a describe request is received.
// This function handles redirections.
func (sh *connectionHandler) OnDescribe(ctx *gortsplib.ServerHandlerOnDescribeCtx) (*base.Response, *gortsplib.ServerStream, error) {
	sh.logger.Debug().Str("client_ip", ctx.Conn.NetConn().RemoteAddr().String()).Msg("rtsp describe request")

	sh.mu.RLock()
	defer sh.mu.RUnlock()

	// Flagship mode:
	if sh.isFlagship {
		paths := strings.Split(ctx.Path, "/")
		if len(paths) < 3 {
			return &base.Response{
				StatusCode: base.StatusConnectionCredentialsNotAccepted,
			}, nil, nil
		}
		streamID := strings.TrimSpace(paths[1])
		passphrase := strings.TrimSpace(paths[2])
		clientIP := ctx.Conn.NetConn().RemoteAddr()

		sh.logger.Debug().Msgf("received onDescribe request from ip '%s' for stream '%s', passphrase '%s'", clientIP.String(), streamID, passphrase)

		if sh.haven.GetStreamID() != streamID {
			sh.logger.Warn().Msgf("stream ID mismatch for client '%s'", clientIP.String())
			return &base.Response{
				StatusCode: base.StatusSessionNotFound,
			}, nil, nil
		}

		if sh.haven.GetPassphrase() != passphrase {
			sh.logger.Warn().Msgf("passphrase mismatch for client '%s'", clientIP.String())
			return &base.Response{
				StatusCode: base.StatusConnectionCredentialsNotAccepted,
			}, nil, nil
		}

		if sh.stream == nil {
			if err := sh.initializeStream(); err != nil {
				sh.logger.Error().Err(err).Msg("failed to start frame writer")
				return &base.Response{
					StatusCode: base.StatusInternalServerError,
				}, nil, nil
			}
		}

		return &base.Response{
			StatusCode: base.StatusOK,
		}, sh.stream, nil

		// escort, err := sh.haven.GetClosestEscort(sh.location)
		// if err != nil {
		// 	if errors.Is(err, types.ErrEscortsNotAvailable) {
		// 		return &base.Response{
		// 			StatusCode: base.StatusOK,
		// 		}, sh.stream, nil
		// 	}
		// 	sh.logger.Error().Err(err).Msgf("failed to get escort for client '%s'", clientIP.String())
		// 	return &base.Response{
		// 		StatusCode: base.StatusInternalServerError,
		// 	}, nil, nil
		// }

		// sh.logger.Info().Msgf("redirecting client '%s' to escort '%s'", clientIP.String(), escort.GetAddr().String())
		// return &base.Response{
		// 	StatusCode: base.StatusMovedPermanently,
		// 	Header: base.Header{
		// 		"Location": base.HeaderValue{"rtsp://" + escort.GetAddr().String() + ctx.Path},
		// 	},
		// }, nil, nil
	}

	// Escort mode:
	if sh.stream == nil {
		return &base.Response{
			StatusCode: base.StatusNotFound,
		}, nil, nil
	}

	return &base.Response{
		StatusCode: base.StatusOK,
	}, sh.stream, nil
}

// OnAnnounce is called when an announce request is received.
// We don't allow publishers, so we just return not implemented.
func (sh *connectionHandler) OnAnnounce(ctx *gortsplib.ServerHandlerOnAnnounceCtx) (*base.Response, error) {
	sh.logger.Warn().Str("client_ip", ctx.Conn.NetConn().RemoteAddr().String()).Msg("client tried to publish data to the stream, which is not supported")

	return &base.Response{
		StatusCode: base.StatusNotImplemented,
	}, nil
}

// OnSetup is called when a setup request is received.
func (sh *connectionHandler) OnSetup(ctx *gortsplib.ServerHandlerOnSetupCtx) (*base.Response, *gortsplib.ServerStream, error) {
	sh.logger.Debug().Str("client_ip", ctx.Conn.NetConn().RemoteAddr().String()).Msg("rtsp setup request")

	if sh.stream == nil {
		return &base.Response{
			StatusCode: base.StatusNotFound,
		}, nil, nil
	}

	return &base.Response{
		StatusCode: base.StatusOK,
	}, sh.stream, nil
}

// OnPlay is called when a play request is received.
func (sh *connectionHandler) OnPlay(ctx *gortsplib.ServerHandlerOnPlayCtx) (*base.Response, error) {
	sh.logger.Info().Str("client_ip", ctx.Conn.NetConn().RemoteAddr().String()).Msg("rtsp play request")

	if sh.stream != nil {
		return &base.Response{
			StatusCode: base.StatusOK,
		}, nil
	}

	return &base.Response{
		StatusCode: base.StatusNotFound,
	}, nil
}

// OnRecord is only called when receiving a frame from a publisher.
// We don't allow publishers, so we just return not implemented.
func (sh *connectionHandler) OnRecord(ctx *gortsplib.ServerHandlerOnRecordCtx) (*base.Response, error) {
	sh.logger.Warn().Str("client_ip", ctx.Conn.NetConn().RemoteAddr().String()).Msg("client tried to push data to the stream via onRecord, which is not supported")

	return &base.Response{
		StatusCode: base.StatusNotImplemented,
	}, nil
}

func (sh *connectionHandler) initializeStream() error {
	bufferStreams, err := sh.haven.GetRTSPStream()
	if err != nil {
		return fmt.Errorf("failed to get RTSP stream from haven: %w", err)
	}

	audioCh, videoCh := make(chan types.Frame, 100), make(chan types.Frame, 100)
	for _, stream := range bufferStreams {
		switch stream.Type {
		case types.BufferTypeAudio:
			audioCh = stream.Channel
		case types.BufferTypeVideo:
			videoCh = stream.Channel
		default:
			sh.logger.Warn().Msgf("received stream with unsupported type '%s', skipping", stream.Type)
		}
	}
	if audioCh == nil || videoCh == nil {
		return errors.New("missing audio or video stream from haven")
	}

	sps, pps, asc, err := sh.extractMetadata(&videoCh, &audioCh)
	if err != nil {
		return fmt.Errorf("failed to extract stream metadata: %w", err)
	}

	err = sh.primeEncode(sps, pps, asc)
	if err != nil {
		return fmt.Errorf("failed to start encoders: %w", err)
	}

	sh.handleFrames(videoCh, 0, sh.encodeH264)
	sh.handleFrames(audioCh, 1, sh.encodeAAC)

	return nil
}

// extractMetadata tries to extracts the SPS/PPS and ASC from each frame until it has all three.
// It'll block until it has all of them.
//
// TODO: add a timeout and handler when the channels are full and we haven't extracted the metadata yet
func (sh *connectionHandler) extractMetadata(videoCh, audioCh *chan types.Frame) (sps, pps []byte, asc *mpeg4audio.AudioSpecificConfig, err error) {
	// exchange channels so that we can peek at the frames to extract SPS/PPS/ASC
	upstreamVideoCh := *videoCh
	upstreamAudioCh := *audioCh
	*videoCh = make(chan types.Frame, cap(upstreamVideoCh))
	*audioCh = make(chan types.Frame, cap(upstreamAudioCh))

	receiveDone := make(chan struct{}, 2)

	sh.wg.Go(func() {
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
						sh.logger.Debug().Err(err).Msg("SPS/PPS not in this frame, continuing")
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
						sh.logger.Debug().Err(err).Msg("ASC not in this frame, continuing")
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

	<-receiveDone
	sh.logger.Debug().Msg("successfully extracted stream metadata, proceeding with stream initialization")
	return sps, pps, asc, nil
}

// primeEncode initializes the stream and encoders with the provided SPS/PPS and ASC.
func (sh *connectionHandler) primeEncode(sps, pps []byte, asc *mpeg4audio.AudioSpecificConfig) error {
	if sh.stream != nil {
		return errors.New("stream already initialized")
	}

	sh.stream = &gortsplib.ServerStream{
		Server: sh.server,
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
						PayloadTyp:       96,
						Config:           asc,
						SizeLength:       13,
						IndexLength:      3,
						IndexDeltaLength: 3,
					}},
				},
			},
		},
	}
	err := sh.stream.Initialize()
	if err != nil {
		return fmt.Errorf("failed to initialize stream: %w", err)
	}

	formatH264, ok := sh.stream.Desc.Medias[0].Formats[0].(*format.H264)
	if !ok {
		return errors.New("failed to assert video format as H264")
	}
	h264Encoder, err := formatH264.CreateEncoder()
	if err != nil {
		return err
	}
	h264Encoder.PayloadMaxSize = 1450

	sh.h264Encoder = h264Encoder
	err = sh.h264Encoder.Init()
	if err != nil {
		return err
	}

	formatAAC, ok := sh.stream.Desc.Medias[1].Formats[0].(*format.MPEG4Audio)
	if !ok {
		return errors.New("failed to assert audio format as MPEG4Audio")
	}
	aacEncoder, err := formatAAC.CreateEncoder()
	if err != nil {
		return err
	}
	sh.aacEncoder = aacEncoder
	return sh.aacEncoder.Init()
}

// handleFrames reads frames from the provided channel, encodes them using the provided encoder function, and writes the resulting RTP packets to the stream.
func (sh *connectionHandler) handleFrames(packetCh <-chan types.Frame, mediaIndex int, encoderFunc func(frame types.Frame) ([]*rtp.Packet, error)) {
	sh.wg.Go(func() {
		for {
			select {
			case frame, ok := <-packetCh:
				if !ok {
					sh.logger.Debug().Msg("frame channel closed, stopping frame writer")
					return
				}

				pkts, err := encoderFunc(frame)
				if err != nil {
					sh.logger.Error().Err(err).Msg("failed to encode frame")
					continue
				}

				sh.mu.RLock()
				media := sh.stream.Desc.Medias[mediaIndex]
				sh.mu.RUnlock()
				for _, pkt := range pkts {
					if err := sh.stream.WritePacketRTP(media, pkt); err != nil {
						sh.logger.Error().Err(err).Msg("failed to write RTP packet")
						continue
					}
				}
			}
		}
	})
}

func (sh *connectionHandler) encodeH264(frame types.Frame) ([]*rtp.Packet, error) {
	defer frame.Decommission()

	var nalus [][]byte
	codec.SplitFrameWithStartCode(frame.Data(), func(nalu []byte) bool {
		naluCopy := make([]byte, len(nalu))
		copy(naluCopy, nalu)
		nalus = append(nalus, naluCopy)
		return true
	})

	if len(nalus) == 0 {
		return nil, errors.New("no NAL units found in frame")
	}

	return sh.h264Encoder.Encode(nalus)
}

func (sh *connectionHandler) encodeAAC(frame types.Frame) ([]*rtp.Packet, error) {
	defer frame.Decommission()
	data := frame.Data()

	adtsHeaderLen := 7
	if len(data) > 1 && (data[1]&0x01) == 0 {
		adtsHeaderLen = 9
	}

	if len(data) < adtsHeaderLen {
		return nil, errors.New("AAC frame too short")
	}

	rawAAC := data[adtsHeaderLen:]
	return sh.aacEncoder.Encode([][]byte{rawAAC})
}
