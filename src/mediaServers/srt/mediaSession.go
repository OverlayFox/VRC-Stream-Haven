package srt

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	srt "github.com/datarhei/gosrt"
	"github.com/datarhei/gosrt/packet"
	"github.com/rs/zerolog"

	"github.com/OverlayFox/VRC-Stream-Haven/src/mediaServers/types"
	globalTypes "github.com/OverlayFox/VRC-Stream-Haven/src/types"
)

var _ globalTypes.MediaSession = (*MediaSession)(nil)

type MediaSession struct {
	Id string

	connection     srt.Conn
	connectionType types.ConnectionType

	packageBuffer globalTypes.PacketBuffer
	stats         *srt.Statistics

	die    sync.Once
	mtx    sync.Mutex
	closed atomic.Bool

	ctx    context.Context
	cancel context.CancelFunc

	logger zerolog.Logger
}

// NewMediaSession creates a new SRT media session
func NewMediaSession(conn srt.Conn, connectionType types.ConnectionType, buffer globalTypes.PacketBuffer, logger zerolog.Logger) globalTypes.MediaSession {
	id := conn.RemoteAddr().String()

	// This context will be replaced by the stream manager, once the session is added to a stream
	ctx, cancel := context.WithCancel(context.Background())

	session := &MediaSession{
		Id: id,

		connection:     conn,
		connectionType: connectionType,

		packageBuffer: buffer,

		ctx:    ctx,
		cancel: cancel,

		logger: logger,
	}
	session.closed.Store(false)

	return session
}

// SetCtx replaces the context of the media session with a new one.
// This is used to set the context of the media session to the stream manager context.
func (s *MediaSession) SetCtx(parentCtx context.Context) {
	s.ctx, s.cancel = context.WithCancel(parentCtx)
}

func (s *MediaSession) GetAddr() net.Addr {
	return s.connection.RemoteAddr()
}

func (s *MediaSession) GetCtx() context.Context {
	return s.ctx
}

func (s *MediaSession) GetId() string {
	return s.Id
}

func (s *MediaSession) IsClosed() bool {
	return s.closed.Load()
}

func (s *MediaSession) IsPublisher() bool {
	return s.connectionType.IsFlagship()
}

func (s *MediaSession) SignalClose() {
	s.cancel()
}

func (s *MediaSession) GetStats() *srt.Statistics {
	s.connection.Stats(s.stats)

	return s.stats
}

func (s *MediaSession) GetRtspPort() (int, error) {
	return -1, fmt.Errorf("RTSP port is not available for SRT connections")
}

// close closes the SRT connection and cleans up the media session.
func (s *MediaSession) close() {
	s.logger.Debug().Msgf("Closing SRT-MediaSession")

	s.die.Do(func() {
		s.cancel()

		s.mtx.Lock()
		defer s.mtx.Unlock()

		if s.connection != nil {
			if err := s.connection.Close(); err != nil {
				s.logger.Error().Err(err).Msg("An error occurred when closing the SRT-MediaSession")
			}
		}

		s.closed.Store(true)
		s.packageBuffer = nil
	})
}

func (s *MediaSession) ReadFromSession() {
	incomingDataChannel := make(chan packet.Packet, 32)
	incomingErrorChannel := make(chan error, 1)

	go func() {
		for {
			select {
			case <-s.ctx.Done():
				return
			default:
				err := s.connection.SetReadDeadline(time.Now().Add(3 * time.Second))
				if err != nil {
					incomingErrorChannel <- err
					continue
				}

				p, err := s.connection.ReadPacket()
				if err != nil {
					incomingErrorChannel <- err
					continue
				}

				select {
				case incomingDataChannel <- p:
				case <-s.ctx.Done():
					return
				}
			}
		}
	}()
	s.logger.Debug().Msg("Started to read data from SRT Flagship")

	for {
		select {
		case <-s.ctx.Done():
			s.close()
			return

		case packet := <-incomingDataChannel:
			if s.IsClosed() || s.packageBuffer == nil {
				s.logger.Debug().Msgf("Session is closed or buffer is nil, cannot write to it")
				packet.Decommission()
				return
			}
			s.packageBuffer.Write(packet)

		case err := <-incomingErrorChannel:
			if errors.Is(err, os.ErrDeadlineExceeded) {
				s.logger.Error().Msg("Read operation on SRT-MediaSession timed out")
				continue
			}

			if errors.Is(err, io.EOF) {
				s.logger.Info().Msg("SRT-MediaSession was closed by the client")
				s.close()
				return
			}

			s.logger.Error().Err(err).Msgf("Failed to receive data from SRT-MediaSession")
			s.SignalClose()
			return
		}
	}
}

func (s *MediaSession) WriteToSession() {
	packetChannel := s.packageBuffer.Subscribe()
	if packetChannel == nil {
		s.logger.Error().Msg("Failed to subscribe to the global packet buffer")
		return
	}
	defer func() {
		if !s.IsClosed() && s.packageBuffer != nil {
			s.packageBuffer.Unsubscribe(packetChannel)
		}
	}()

	for {
		select {
		case <-s.ctx.Done():
			s.close()
			return
		case packet, ok := <-packetChannel:
			if !ok {
				s.logger.Error().Msg("Packet channel was closed")
				packet.Decommission()
				return
			}

			if s.IsClosed() || s.packageBuffer == nil {
				s.logger.Error().Msg("SRT-MediaSession is closed, cannot write to it")
				packet.Decommission()
				return
			}

			if err := s.connection.SetWriteDeadline(time.Now().Add(2 * time.Second)); err != nil {
				s.logger.Warn().Err(err).Msg("Failed to set write deadline")
				packet.Decommission()
				continue
			}

			err := s.connection.WritePacket(packet)
			if err != nil {
				if errors.Is(err, io.EOF) {
					s.logger.Info().Msg("SRT-MediaSession was closed by the client")
					packet.Decommission()
					s.close()
					return
				}

				s.logger.Error().Err(err).Msg("Failed to write packet to SRT-MediaSession")
				packet.Decommission()
				continue
			}
			packet.Decommission()
		}
	}
}
