package rtsp

import (
	"context"
	"fmt"
	"sync"

	"github.com/OverlayFox/VRC-Stream-Haven/src/types"
	"github.com/bluenviron/gortsplib/v5"
	"github.com/bluenviron/gortsplib/v5/pkg/base"
	"github.com/rs/zerolog"
)

type Server struct {
	logger zerolog.Logger
	config Config

	server *gortsplib.Server

	sessions   map[*gortsplib.ServerSession]types.RTSPConnection
	sessionMtx sync.RWMutex

	haven   types.Haven
	locator types.Locator

	isFlagship bool

	ctx    context.Context
	cancel context.CancelFunc
	mtx    sync.RWMutex
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

		sessions: make(map[*gortsplib.ServerSession]types.RTSPConnection),

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

func (s *Server) removeConn(session *gortsplib.ServerSession) {
	s.sessionMtx.Lock()
	delete(s.sessions, session)
	s.sessionMtx.Unlock()
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

// monitorConn adds the connection to the session map and starts a goroutine to monitor its context for cancellation.
func (s *Server) monitorConn(session *gortsplib.ServerSession, connection types.RTSPConnection) {
	s.sessionMtx.Lock()
	s.sessions[session] = connection
	s.sessionMtx.Unlock()

	s.wg.Go(func() {
		connLogger := connection.GetLogger()
		select {
		case <-connection.GetCtx().Done():
			connLogger.Debug().Str("client_ip", connection.GetAddr().String()).Msg("connection context done, closing connection")
		case <-s.ctx.Done():
			connLogger.Debug().Str("client_ip", connection.GetAddr().String()).Msg("server context done, closing connection")
		}

		connection.Close()
		s.removeConn(session)
	})
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
	s.logger.Debug().Str("client_ip", ctx.Conn.NetConn().RemoteAddr().String()).Msg("RTSP client disconnected on request from client")
}

// OnSessionOpen is called when a session is opened.
func (s *Server) OnSessionOpen(ctx *gortsplib.ServerHandlerOnSessionOpenCtx) {
	conn := ctx.Conn.NetConn()
	location, err := s.locator.GetLocation(conn.RemoteAddr())
	if err != nil {
		location = types.Location{}
	}

	connection := NewConnection(s.logger, s.ctx, conn, location, s.server)
	connLogger := connection.GetLogger()
	connLogger.Info().Msg("RTSP session opened")
	s.monitorConn(ctx.Conn.Session(), connection)
}

// OnSessionClose is called when a session is closed.
func (s *Server) OnSessionClose(ctx *gortsplib.ServerHandlerOnSessionCloseCtx) {
	s.logger.Info().Msg("RTSP session closed on request from client")

	s.sessionMtx.RLock()
	connection, ok := s.sessions[ctx.Session]
	s.sessionMtx.RUnlock()
	if ok {
		connection.Close()
	}
}

// OnSetup is called when a setup request is received.
func (s *Server) OnSetup(ctx *gortsplib.ServerHandlerOnSetupCtx) (*base.Response, *gortsplib.ServerStream, error) {
	s.logger.Debug().Str("client_ip", ctx.Conn.NetConn().RemoteAddr().String()).Msg("rtsp setup request")

	s.sessionMtx.RLock()
	connection, ok := s.sessions[ctx.Conn.Session()]
	s.sessionMtx.RUnlock()
	if !ok {
		return &base.Response{StatusCode: base.StatusNotFound}, nil, nil
	}

	stream := connection.GetStream()
	if stream == nil {
		return &base.Response{StatusCode: base.StatusNotFound}, nil, nil
	}

	return &base.Response{StatusCode: base.StatusOK}, stream, nil
}

// OnDescribe is called when a describe request is received.
// This function handles redirections.
func (s *Server) OnDescribe(ctx *gortsplib.ServerHandlerOnDescribeCtx) (*base.Response, *gortsplib.ServerStream, error) {
	s.logger.Debug().Str("client_ip", ctx.Conn.NetConn().RemoteAddr().String()).Msg("rtsp describe request")

	s.mtx.RLock()
	defer s.mtx.RUnlock()

	s.sessionMtx.RLock()
	connection := s.sessions[ctx.Conn.Session()]
	s.sessionMtx.RUnlock()

	connLogger := connection.GetLogger()
	clientIP := ctx.Conn.NetConn().RemoteAddr()

	if clientIP != connection.GetAddr() {
		connLogger.Warn().Str("client_ip", clientIP.String()).Msg("client IP mismatch between connection and describe request")
		return &base.Response{StatusCode: base.StatusBadRequest}, nil, nil
	}

	streamID, passphrase, err := GetCredentials(ctx.Path)
	if err != nil {
		connLogger.Info().Err(err).Msgf("Invalid describe path '%s'", ctx.Path)
		return &base.Response{StatusCode: base.StatusConnectionCredentialsNotAccepted}, nil, nil
	}

	if s.haven.GetStreamID() != streamID {
		connLogger.Info().Msg("Invalid stream ID in describe request")
		return &base.Response{StatusCode: base.StatusSessionNotFound}, nil, nil
	}

	if s.haven.GetPassphrase() != passphrase {
		connLogger.Warn().Msg("Invalid passphrase in describe request")
		return &base.Response{StatusCode: base.StatusConnectionCredentialsNotAccepted}, nil, nil
	}

	if s.isFlagship {
		bufferStreams, err := s.haven.GetRTSPStream()
		if err != nil {
			connLogger.Error().Err(err).Msg("failed to get stream from haven")
			return &base.Response{StatusCode: base.StatusInternalServerError}, nil, nil
		}

		err = connection.Write(bufferStreams)
		if err != nil {
			connLogger.Error().Err(err).Msg("failed to write to connection")
			return &base.Response{StatusCode: base.StatusInternalServerError}, nil, nil
		}

		stream := connection.GetStream()
		if stream == nil {
			connLogger.Error().Msg("stream not initialized for connection")
			return &base.Response{StatusCode: base.StatusInternalServerError}, nil, nil
		}
		return &base.Response{StatusCode: base.StatusOK}, stream, nil

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
	// if sh.stream == nil {
	// 	return &base.Response{
	// 		StatusCode: base.StatusNotFound,
	// 	}, nil, nil
	// }

	return &base.Response{
		StatusCode: base.StatusInternalServerError,
	}, nil, nil
}

// OnPlay is called when a play request is received.
func (s *Server) OnPlay(ctx *gortsplib.ServerHandlerOnPlayCtx) (*base.Response, error) {
	s.logger.Info().Str("client_ip", ctx.Conn.NetConn().RemoteAddr().String()).Msg("RTSP client play request")

	s.sessionMtx.RLock()
	connection, ok := s.sessions[ctx.Conn.Session()]
	s.sessionMtx.RUnlock()
	if !ok {
		return &base.Response{StatusCode: base.StatusNotFound}, nil
	}

	err := connection.StartPlay()
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
