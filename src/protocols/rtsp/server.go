package rtsp

import (
	"fmt"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/OverlayFox/VRC-Stream-Haven/src/types"
	"github.com/bluenviron/gortsplib/v5"
	"github.com/rs/zerolog"
)

type RtspConfig struct {
	Port       int
	Address    net.Addr
	Passphrase string
	IsFlagship bool
}

func (c *RtspConfig) Validate() error {
	if c.Port < 0 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d", c.Port)
	}

	if len(c.Passphrase) < 10 {
		return fmt.Errorf("passphrase must be at least 10 characters long")
	}

	if url.PathEscape(c.Passphrase) != c.Passphrase {
		return fmt.Errorf("passphrase contains characters that are not safe for a URL path")
	}

	return nil
}

type Server struct {
	logger zerolog.Logger
	config RtspConfig

	handler *serverHandler

	wg sync.WaitGroup
}

func New(config RtspConfig, haven types.Haven, logger zerolog.Logger) (types.ProtocolServer, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	h := &serverHandler{
		logger:     logger,
		isFlagship: true,
		haven:      haven,
	}
	h.server = &gortsplib.Server{
		Handler:     h,
		RTSPAddress: fmt.Sprintf(":%d", config.Port),

		// MaxPacketSize: 600,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		// WriteQueueSize: 256 * 4,
	}

	return &Server{
		logger:  logger,
		config:  config,
		handler: h,
	}, nil
}

func (s *Server) Start() {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.handler.server.StartAndWait(); err != nil {
			s.logger.Error().Err(err).Msg("rtsp server stopped")
		}
	}()
	s.logger.Info().Msgf("started rtsp listener on tcp address '%s'", s.handler.server.RTSPAddress)
}

func (s *Server) Close() {
	s.handler.server.Close()
	s.wg.Wait()
}
