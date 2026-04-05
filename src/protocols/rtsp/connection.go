package rtsp

import (
	"errors"
	"strings"
	"sync"

	"github.com/OverlayFox/VRC-Stream-Haven/src/types"
	"github.com/bluenviron/gortsplib/v5"
	"github.com/bluenviron/gortsplib/v5/pkg/base"
	"github.com/bluenviron/gortsplib/v5/pkg/description"
	"github.com/bluenviron/gortsplib/v5/pkg/format"
	"github.com/bluenviron/gortsplib/v5/pkg/format/rtph264"
	"github.com/bluenviron/mediacommon/v2/pkg/codecs/mpeg4audio"
	"github.com/rs/zerolog"
)

type Connection struct {
	logger zerolog.Logger

	server *gortsplib.Server
	stream *gortsplib.ServerStream

	haven types.Haven

	isFlagship bool
	rtpEncoder *rtph264.Encoder

	mu sync.RWMutex
}

// OnConnOpen is called when a connection is opened.
func (sh *Connection) OnConnOpen(ctx *gortsplib.ServerHandlerOnConnOpenCtx) {
	sh.logger.Info().Str("client_ip", ctx.Conn.NetConn().RemoteAddr().String()).Msg("rtsp connection opened")
}

// OnConnClose is called when a connection is closed.
func (sh *Connection) OnConnClose(ctx *gortsplib.ServerHandlerOnConnCloseCtx) {
	sh.logger.Debug().Str("client_ip", ctx.Conn.NetConn().RemoteAddr().String()).Msgf("rtsp connection closed: %v", ctx.Error)
}

// OnSessionOpen is called when a session is opened.
func (sh *Connection) OnSessionOpen(ctx *gortsplib.ServerHandlerOnSessionOpenCtx) {
	sh.logger.Debug().Str("client_ip", ctx.Conn.NetConn().RemoteAddr().String()).Msg("rtsp session opened")
}

// OnSessionClose is called when a session is closed.
func (sh *Connection) OnSessionClose(_ *gortsplib.ServerHandlerOnSessionCloseCtx) {
	sh.logger.Debug().Msg("rtsp session closed")
}

// OnDescribe is called when a describe request is received.
// This function handles redirections.
func (sh *Connection) OnDescribe(ctx *gortsplib.ServerHandlerOnDescribeCtx) (*base.Response, *gortsplib.ServerStream, error) {
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

		sh.logger.Debug().Msgf("received read request from ip '%s' for stream '%s', passphrase '%s'", clientIP.String(), streamID, passphrase)

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
			sh.logger.Info().Msgf("stream does not exist yet for client '%s'", clientIP.String())
			return &base.Response{
				StatusCode: base.StatusNotFound,
			}, nil, nil
		}

		escort, err := sh.haven.GetClosestEscort(clientIP)
		if err != nil {
			if errors.Is(err, types.ErrEscortsNotAvailable) {
				return &base.Response{
					StatusCode: base.StatusOK,
				}, sh.stream, nil
			}
			sh.logger.Error().Err(err).Msgf("failed to get escort for client '%s'", clientIP.String())
			return &base.Response{
				StatusCode: base.StatusInternalServerError,
			}, nil, nil
		}

		sh.logger.Info().Msgf("redirecting client '%s' to escort '%s'", clientIP.String(), escort.GetAddr().String())
		return &base.Response{
			StatusCode: base.StatusMovedPermanently,
			Header: base.Header{
				"Location": base.HeaderValue{"rtsp://" + escort.GetAddr().String() + ctx.Path},
			},
		}, nil, nil
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
func (sh *Connection) OnAnnounce(ctx *gortsplib.ServerHandlerOnAnnounceCtx) (*base.Response, error) {
	sh.logger.Warn().Str("client_ip", ctx.Conn.NetConn().RemoteAddr().String()).Msg("client tried to publish data to the stream, which is not supported")

	return &base.Response{
		StatusCode: base.StatusNotImplemented,
	}, nil
}

// OnSetup is called when a setup request is received.
func (sh *Connection) OnSetup(ctx *gortsplib.ServerHandlerOnSetupCtx) (*base.Response, *gortsplib.ServerStream, error) {
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
func (sh *Connection) OnPlay(ctx *gortsplib.ServerHandlerOnPlayCtx) (*base.Response, error) {
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
func (sh *Connection) OnRecord(ctx *gortsplib.ServerHandlerOnRecordCtx) (*base.Response, error) {
	sh.logger.Warn().Str("client_ip", ctx.Conn.NetConn().RemoteAddr().String()).Msg("client tried to push data to the stream via onRecord, which is not supported")

	return &base.Response{
		StatusCode: base.StatusNotImplemented,
	}, nil
}

func (sh *Connection) WritePacketRTP(frame types.Frame) error {
	if sh.stream == nil {
		sh.stream = &gortsplib.ServerStream{
			Server: sh.server,
			Desc: &description.Session{
				Medias: []*description.Media{
					{
						Type: description.MediaTypeVideo,
						Formats: []format.Format{&format.H264{
							PayloadTyp:        96,
							PacketizationMode: 1,
						}},
					},
					{
						Type: description.MediaTypeAudio,
						Formats: []format.Format{&format.MPEG4Audio{
							PayloadTyp: 96,
							Config: &mpeg4audio.AudioSpecificConfig{
								Type:         mpeg4audio.ObjectTypeAACLC,
								SampleRate:   48000,
								ChannelCount: 2,
							},
						}}, // AAC
					},
				},
			},
		}
	}
	// h264Encoder, err := sh.stream.Desc.Medias[0].Formats[0].(*format.H264).CreateEncoder()
	// if err != nil {
	// 	return err
	// }
	// aacEncoder, err := sh.stream.Desc.Medias[1].Formats[0].(*format.MPEG4Audio).CreateEncoder()
	// if err != nil {
	// 	return err
	// }

	// return sh.stream.WritePacketRTP()

	return nil
}
