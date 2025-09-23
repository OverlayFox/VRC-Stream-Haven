package srt

import (
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/OverlayFox/VRC-Stream-Haven/src/types"
	goSrt "github.com/datarhei/gosrt"
	"github.com/rs/zerolog"
)

type SrtConfig struct {
	Port       int
	Address    net.Addr
	Passphrase string
	IsFlagship bool
}

type SRTServer struct {
	logger zerolog.Logger

	listener goSrt.Listener
	haven    types.Haven
	config   goSrt.Config

	die  sync.Once
	quit chan struct{}
}

var _ types.ProtocolServer = (*SRTServer)(nil)

func New(config SrtConfig, haven types.Haven, logger zerolog.Logger) (types.ProtocolServer, error) {
	listener, err := goSrt.Listen("srt", fmt.Sprintf(":%d", config.Port), goSrt.DefaultConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to start srt listener: %w", err)
	}

	return &SRTServer{
		logger: logger,

		listener: listener,
		haven:    haven,
		config:   goSrt.DefaultConfig(),

		quit: make(chan struct{}),
	}, nil
}

// Start start the SRT server and blocks until it is closed or an error occurs.
func (s *SRTServer) Start() {
	acceptCh := make(chan goSrt.ConnRequest)
	errCh := make(chan error)

	// listener runs in a separate goroutine to accept incoming connections
	go func() {
		for {
			req, err := s.listener.Accept2()
			if err != nil {
				if errors.Is(err, goSrt.ErrListenerClosed) {
					s.logger.Info().Msg("srt listener has been closed")
					return
				}
				errCh <- err
				return
			}
			acceptCh <- req
		}
	}()
	s.logger.Info().Msgf("started srt listener on tcp address '%s'", s.listener.Addr().String())

	connectionConfig := ConnectionConfig{
		ReadTimeout:  goSrt.DefaultConfig().ConnectionTimeout,
		WriteTimeout: goSrt.DefaultConfig().ConnectionTimeout,
	}

	go func() {
		for {
			select {
			case <-s.quit:
				return

			case req := <-acceptCh:
				_, err := NewConnection(s.logger, req, s.haven, connectionConfig)
				if err != nil {
					s.logger.Error().Err(err).Msg("failed to create new srt connection")
					continue
				}

			case err := <-errCh:
				s.logger.Error().Err(err).Msg("srt listener error")
			}
		}
	}()
}

func (s *SRTServer) Close() {
	s.logger.Info().Msg("Closing SRT server")
	s.listener.Close()
	close(s.quit)

	s.listener = nil
}
