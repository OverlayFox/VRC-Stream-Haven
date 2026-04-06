package srt

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/OverlayFox/VRC-Stream-Haven/src/types"
	goSrt "github.com/datarhei/gosrt"
	"github.com/rs/zerolog"
)

type Config struct {
	Address string
	Port    int
}

type server struct {
	logger zerolog.Logger
	config Config

	haven   types.Haven
	locator types.Locator

	listener goSrt.Listener

	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

func New(upstreamCtx context.Context, logger zerolog.Logger, config Config, haven types.Haven, locator types.Locator) (types.ProtocolServer, error) {
	listener, err := goSrt.Listen("srt", fmt.Sprintf("%s:%d", config.Address, config.Port), goSrt.DefaultConfig()) // TODO: Optimise DefaultConfig() for large WAN hops
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(upstreamCtx)
	s := &server{
		logger: logger.With().Str("protocol", "srt").Logger(),
		config: config,

		haven:   haven,
		locator: locator,

		listener: listener,

		ctx:    ctx,
		cancel: cancel,
	}
	return s, nil
}

func (s *server) Start() {
	acceptCh := make(chan goSrt.ConnRequest, 10)
	errCh := make(chan error, 1)

	// start listener
	s.wg.Go(func() {
		for {
			req, err := s.listener.Accept2()
			if err != nil {
				if errors.Is(err, goSrt.ErrListenerClosed) {
					s.logger.Info().Msg("SRT listener closed")
					return
				}
				errCh <- err
				return
			}
			acceptCh <- req
		}
	})
	s.logger.Info().Msgf("Started SRT listener on address '%s:%d'", s.config.Address, s.config.Port)

	// handle incoming connections
	s.wg.Go(func() {
		for {
			select {
			case <-s.ctx.Done():
				return
			case err := <-errCh:
				s.logger.Error().Err(err).Msg("SRT listener error on connection accept")
			case req := <-acceptCh:
				connectionLogger := s.logger.With().Str("remote_addr", req.RemoteAddr().String()).Str("stream_id", req.StreamId()).Logger()
				connectionLogger.Info().Msg("Accepted new SRT connection")

				go func() {
					conn, err := NewConnection(s.ctx, connectionLogger, s.haven, s.locator, req)
					if err != nil {
						connectionLogger.Error().Err(err).Msg("Failed to handle SRT connection")
						return
					}
					if err := s.haven.AddConnection(conn); err != nil {
						connectionLogger.Error().Err(err).Msg("Failed to add SRT connection to haven")
						conn.Close()
						return
					}
					connectionLogger.Info().Msg("SRT connection added to haven")
				}()
			}
		}
	})
}

func (s *server) Close() {
	s.listener.Close()
	s.cancel()
	s.wg.Wait()

	s.logger.Info().Msg("SRT server closed")
}
