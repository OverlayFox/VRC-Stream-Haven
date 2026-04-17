package rtsp

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/bluenviron/gortsplib/v5"
	"github.com/bluenviron/gortsplib/v5/pkg/base"
	"github.com/rs/zerolog"

	"github.com/OverlayFox/VRC-Stream-Haven/src/types"
)

type Server struct {
	logger zerolog.Logger
	config Config

	server *gortsplib.Server

	haven   types.Haven
	locator types.Locator

	isFlagship bool

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func New(upstreamCtx context.Context, logger zerolog.Logger, config Config, haven types.Haven, locator types.Locator) (types.ProtocolServer, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(upstreamCtx)
	s := &Server{
		logger: logger,
		config: config,

		haven:      haven,
		locator:    locator,
		isFlagship: config.IsFlagship,

		ctx:    ctx,
		cancel: cancel,
	}
	s.server = &gortsplib.Server{
		Handler:     s,
		RTSPAddress: fmt.Sprintf("%s:%d", config.Address, config.Port),

		ReadTimeout:    config.ReadTimeout,
		WriteTimeout:   config.WriteTimeout,
		MaxPacketSize:  config.MaxPacketSize,
		WriteQueueSize: config.WriteQueueSize,
	}

	return s, nil
}

func (s *Server) Dial(address string, streamID, passphrase string) error {
	return errors.New("dialing is not supported on RTSP server")
}

func (s *Server) Start() {
	s.wg.Go(func() {
		if err := s.server.StartAndWait(); err != nil {
			s.logger.Error().Err(err).Msg("rtsp server stopped")
		}
	})
	s.logger.Info().Msgf("Started rtsp listener on tcp address '%s'", s.server.RTSPAddress)
}

func (s *Server) Close() {
	s.cancel()
	s.wg.Wait()

	s.logger.Info().Msg("RTSP server stopped")
}

func (s *Server) validate(path string) (*base.Response, error) {
	streamID, passphrase, err := GetCredentials(path)
	if err != nil {
		return &base.Response{StatusCode: base.StatusConnectionCredentialsNotAccepted}, errors.New("invalid describe path")
	}
	if s.haven.GetStreamID() != streamID {
		return &base.Response{StatusCode: base.StatusSessionNotFound}, errors.New("invalid stream ID")
	}
	if s.haven.GetPassphrase() != passphrase {
		return &base.Response{StatusCode: base.StatusConnectionCredentialsNotAccepted}, errors.New("invalid passphrase")
	}
	return &base.Response{StatusCode: base.StatusOK}, nil
}

//
// Callback functions
//

// OnConnOpen is called when a connection is opened.
func (s *Server) OnConnOpen(ctx *gortsplib.ServerHandlerOnConnOpenCtx) {
	s.logger.Debug().Str("client_ip", ctx.Conn.NetConn().RemoteAddr().String()).Msg("RTSP client connected")
}

// OnConnClose is called when a connection is closed.
func (s *Server) OnConnClose(ctx *gortsplib.ServerHandlerOnConnCloseCtx) {
	s.logger.Debug().Str("client_ip", ctx.Conn.NetConn().RemoteAddr().String()).Msg("RTSP client disconnected")
}

// OnDescribe is called when a describe request is received.
// This function handles redirections.
func (s *Server) OnDescribe(ctx *gortsplib.ServerHandlerOnDescribeCtx) (*base.Response, *gortsplib.ServerStream, error) {
	conn := ctx.Conn.NetConn()
	location, err := s.locator.GetLocation(conn.RemoteAddr())
	if err != nil {
		location = types.Location{}
	}
	connection := NewConnection(s.logger, s.ctx, conn, location, s.server, ctx.Conn.Session())
	connLogger := connection.GetLogger()
	connLogger.Info().Msg("RTSP describe request received")

	response, err := s.validate(ctx.Path)
	if err != nil {
		connLogger.Info().Err(err).Msg("RTSP describe request validation failed")
		return response, nil, nil
	}

	if s.isFlagship {
		escort := s.haven.GetClosestEscort(connection.GetLocation())
		if escort != nil {
			connLogger.Info().Msgf("Redirecting client to escort '%s'", escort.GetLocation().String())
			return &base.Response{
				StatusCode: base.StatusMovedPermanently,
				Header: base.Header{
					"Location": base.HeaderValue{"rtsp://" + escort.GetAddr().String() + ctx.Path},
				},
			}, nil, nil
		}
	}

	err = s.haven.AddConnection(connection)
	if err != nil {
		connLogger.Error().Err(err).Msg("Failed to add connection to haven")
		return &base.Response{StatusCode: base.StatusInternalServerError}, nil, nil
	}

	stream := connection.GetStream()
	if stream == nil {
		connLogger.Error().Msg("stream not initialized for connection")
		return &base.Response{StatusCode: base.StatusInternalServerError}, nil, nil
	}
	return &base.Response{StatusCode: base.StatusOK}, stream, nil
}

// OnSessionOpen is called when a session is opened.
func (s *Server) OnSessionOpen(ctx *gortsplib.ServerHandlerOnSessionOpenCtx) {
	s.logger.Debug().Str("client_ip", ctx.Conn.NetConn().RemoteAddr().String()).Msg("RTSP session opened")
}

// OnSessionClose is called when a session is closed.
func (s *Server) OnSessionClose(ctx *gortsplib.ServerHandlerOnSessionCloseCtx) {
	s.logger.Debug().Msg("RTSP session closed on request from client")
}

// OnSetup is called when a setup request is received.
func (s *Server) OnSetup(ctx *gortsplib.ServerHandlerOnSetupCtx) (*base.Response, *gortsplib.ServerStream, error) {
	connection, err := s.haven.GetViewer(ctx.Conn.NetConn().RemoteAddr())
	if err != nil {
		s.logger.Error().Err(err).Msgf("Failed to get viewer for client '%s'", ctx.Conn.NetConn().RemoteAddr().String())
		return &base.Response{StatusCode: base.StatusConnectionAuthorizationRequired}, nil, nil
	}
	connLogger := connection.GetLogger()
	connLogger.Info().Msg("RTSP client setup request received")

	return &base.Response{StatusCode: base.StatusOK}, connection.GetStream(), nil
}

// OnPlay is called when a play request is received.
func (s *Server) OnPlay(ctx *gortsplib.ServerHandlerOnPlayCtx) (*base.Response, error) {
	connection, err := s.haven.GetViewer(ctx.Conn.NetConn().RemoteAddr())
	if err != nil {
		s.logger.Error().Err(err).Msgf("Failed to get viewer for client '%s'", ctx.Conn.NetConn().RemoteAddr().String())
		return &base.Response{StatusCode: base.StatusConnectionAuthorizationRequired}, nil
	}
	connLogger := connection.GetLogger()
	connLogger.Info().Msg("RTSP client play request received")

	err = connection.StartPlay()
	if err != nil {
		connLogger := connection.GetLogger()
		connLogger.Error().Err(err).Msg("Failed to start RTSP client playback")
		return &base.Response{StatusCode: base.StatusInternalServerError}, nil
	}

	return &base.Response{StatusCode: base.StatusOK}, nil
}

// OnAnnounce is called when an announce request is received.
// We don't allow publishers, so we just return forbidden.
func (s *Server) OnAnnounce(ctx *gortsplib.ServerHandlerOnAnnounceCtx) (*base.Response, error) {
	s.logger.Warn().Str("client_ip", ctx.Conn.NetConn().RemoteAddr().String()).Msg("RTSP client tried to publish data to the stream")
	return &base.Response{StatusCode: base.StatusForbidden}, nil
}

// OnRecord is only called when receiving a frame from a publisher.
// We don't allow publishers, so we just return forbidden.
func (s *Server) OnRecord(ctx *gortsplib.ServerHandlerOnRecordCtx) (*base.Response, error) {
	s.logger.Warn().Str("client_ip", ctx.Conn.NetConn().RemoteAddr().String()).Msg("RTSP client tried to publish data to the stream")
	return &base.Response{StatusCode: base.StatusForbidden}, nil
}
