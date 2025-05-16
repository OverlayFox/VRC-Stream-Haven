package rtsp

import (
	"fmt"
	"sync"
	"time"

	globalTypes "github.com/OverlayFox/VRC-Stream-Haven/src/types"
	"github.com/bluenviron/gortsplib/v4"
	"github.com/rs/zerolog"
)

type RTSPServer struct {
	listener *gortsplib.Server

	isFlagship bool

	logger zerolog.Logger
	die    sync.Once
	quit   chan struct{}
}

func New(governor globalTypes.Governor, port int, isFlagship bool, passphrase string, logger zerolog.Logger) (globalTypes.ProtocolServer, error) {
	if port < 0 || port > 65535 {
		return nil, fmt.Errorf("invalid port: %d", port)
	}

	rtspServer := &RTSPServer{
		isFlagship: isFlagship,

		logger: logger,
		die:    sync.Once{},
		quit:   make(chan struct{}),
	}

	rtspServer.listener = &gortsplib.Server{
		RTSPAddress: fmt.Sprintf(":%d", port),

		MaxPacketSize:  600, // kept small to avoid fragmentation over WAN
		ReadTimeout:    5 * time.Second,
		WriteTimeout:   2 * time.Second,
		WriteQueueSize: 256 * 4, // amount of packets the buffer can hold
	}
	if isFlagship {
		handler := NewFlagshipHandler(rtspServer.listener, governor, passphrase, logger.With().Bool("is_flagship", true).Logger())
		rtspServer.listener.Handler = &handler
	} else {
		handler := NewHandler(rtspServer.listener, governor, passphrase, logger.With().Bool("is_flagship", false).Logger())
		rtspServer.listener.Handler = &handler
	}

	return rtspServer, nil
}

// Start starts the RTSP server and blocks until it is closed or an error occurs.
func (s *RTSPServer) Start() error {
	err := s.listener.Start()
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to start RTSP listener")
		s.Close()
		return err
	}
	s.logger.Info().Msgf("Started RTSP listener on TCP address '%s'", s.listener.RTSPAddress)

	err = s.listener.Wait()
	if err != nil {
		s.logger.Error().Err(err).Msg("RTSP listener stopped")
		return err
	}

	return nil
}

func (s *RTSPServer) Close() {
	s.die.Do(func() {
		close(s.quit)
		s.listener.Close()

		s.listener = nil

		s.logger.Info().Msg("RTSP server closed")
	})
}
