package rtsp

import (
	"context"
	"fmt"
	"sync"

	"github.com/OverlayFox/VRC-Stream-Haven/src/types"
	"github.com/bluenviron/gortsplib/v5"
	"github.com/rs/zerolog"
)

type Server struct {
	logger zerolog.Logger
	config Config

	handler *Connection

	wg sync.WaitGroup
}

func New(upstreamCtx context.Context, logger zerolog.Logger, config Config, haven types.Haven, locator types.Locator) (types.ProtocolServer, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	h := &Connection{
		logger:     logger,
		isFlagship: config.IsFlagship,
		haven:      haven,
		locator:    locator,
	}
	h.server = &gortsplib.Server{
		Handler:     h,
		RTSPAddress: fmt.Sprintf("%s:%d", config.Address, config.Port),

		ReadTimeout:    config.ReadTimeout,
		WriteTimeout:   config.WriteTimeout,
		MaxPacketSize:  config.MaxPacketSize,
		WriteQueueSize: config.WriteQueueSize,
	}

	return &Server{
		logger:  logger,
		config:  config,
		handler: h,
	}, nil
}

func (s *Server) Start() {
	s.wg.Go(func() {
		if err := s.handler.server.StartAndWait(); err != nil {
			s.logger.Error().Err(err).Msg("rtsp server stopped")
		}
	})
	s.logger.Info().Msgf("Started rtsp listener on tcp address '%s'", s.handler.server.RTSPAddress)
}

func (s *Server) Close() {
	s.handler.server.Close()
	s.wg.Wait()
}
