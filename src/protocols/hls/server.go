package hls

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/rs/zerolog"

	"github.com/OverlayFox/VRC-Stream-Haven/src/types"
)

type Server struct {
	logger zerolog.Logger
	config Config

	httpServer *http.Server

	haven   types.Haven
	locator types.Locator

	isFlagship bool

	connection types.ConnectionRTSP
	connMtx    sync.Mutex

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
	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", config.Address, config.Port),
		Handler: s,
	}

	return s, nil
}

func (s *Server) Dial(address, streamId, passphrase string) error {
	return errors.New("HLS server does not support dialing out to other servers")
}

func (s *Server) Start() {
	s.wg.Go(func() {
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error().Err(err).Msg("HLS server failed to start")
		}
	})
	s.logger.Info().Msgf("HLS server started on %s:%d", s.config.Address, s.config.Port)
}

func (s *Server) Close() {
	s.cancel()

	_ = s.httpServer.Shutdown(context.Background())
	s.wg.Wait()

	s.connMtx.Lock()
	if s.connection != nil {
		s.connection.Close()
	}
	s.connMtx.Unlock()

	s.logger.Info().Msg("HLS server stopped")
}

func (s *Server) validate(path string) (int, error) {
	streamID, passphrase, err := GetCredentials(path)
	if err != nil {
		return http.StatusBadRequest, errors.New("invalid stream path")
	}
	if s.haven.GetStreamID() != streamID {
		return http.StatusNotFound, errors.New("invalid stream ID")
	}
	if s.haven.GetPassphrase() != passphrase {
		return http.StatusUnauthorized, errors.New("invalid passphrase")
	}
	return http.StatusOK, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*") // Allow CORS for all origins

	status, err := s.validate(r.URL.Path)
	if err != nil {
		s.logger.Info().Err(err).Str("client_ip", r.RemoteAddr).Msg("HLS request validation failed")
		http.Error(w, err.Error(), status)
		return
	}

	var location types.Location
	if s.isFlagship {
		addr, err := net.ResolveTCPAddr("tcp", r.RemoteAddr)
		if err != nil {
			s.logger.Error().Err(err).Str("client_ip", r.RemoteAddr).Msg("Failed to resolve client address")
			http.Error(w, "Failed to resolve client address", http.StatusInternalServerError)
			return
		}

		location, err = s.locator.GetLocation(addr)
		if err == nil {
			escort := s.haven.GetClosestEscort(location)
			if escort != nil {
				s.logger.Info().Msgf("Redirecting client to escort '%s'", escort.GetLocation().String())
				http.Redirect(w, r, "http://"+escort.GetAddr().String()+r.URL.Path, http.StatusMovedPermanently)
				return
			}
		}
	}

	s.connMtx.Lock()
	if s.connection == nil {
		s.logger.Info().Msg("First LL-HLS viewer connected, priming stream muxer...")
		s.connection = NewConnection(s.ctx, s.logger, location)
		if err := s.haven.AddConnection(s.connection); err != nil {
			s.connMtx.Unlock()
			s.logger.Error().Err(err).Msg("Failed to hook HLS connection to stream haven")
			http.Error(w, "Failed to initialize stream", http.StatusInternalServerError)
			return
		}
	}
	s.connMtx.Unlock()

	// Push the HLS request down to the underlying `gohlslib.Muxer`
	s.connection.HandleHTTP(w, r)
}
