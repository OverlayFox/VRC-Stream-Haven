package srt

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	goSrt "github.com/datarhei/gosrt"
	"github.com/datarhei/gosrt/packet"
	"github.com/rs/zerolog"

	"github.com/OverlayFox/VRC-Stream-Haven/src/multiplexer"
	"github.com/OverlayFox/VRC-Stream-Haven/src/types"
)

const (
	initialBackoff    = 100 * time.Millisecond
	maxBackoff        = 30 * time.Second
	backoffMultiplier = 2.0
)

type connection struct {
	logger zerolog.Logger

	config   goSrt.Config
	conn     goSrt.Conn
	connType types.ConnectionType
	isEscort bool

	demuxer *multiplexer.MpegTsDemuxer

	haven    types.Haven
	location types.Location

	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

func NewConnection(upstreamCtx context.Context, logger zerolog.Logger, haven types.Haven, locator types.Locator, conn goSrt.Conn, config goSrt.Config) (types.ConnectionSRT, error) {
	logger = logger.With().Str("type", "srt").Logger() // TODO: add IP and location once we have them
	ctx, cancel := context.WithCancel(upstreamCtx)
	c := &connection{
		logger: logger,

		config:   config,
		conn:     conn,
		connType: types.ConnectionTypePublisher,
		isEscort: true,

		demuxer: multiplexer.NewMpegTsDemuxer(ctx, logger.With().Str("component", "demuxer").Logger(), multiplexer.Settings{
			InputBufferCap:  50,
			OutputBufferCap: 200,
			AudioDriftLimit: 20 * time.Millisecond,
		}),

		haven: haven,

		ctx:    ctx,
		cancel: cancel,
	}

	return c, nil
}

func NewConnectionFromRequest(upstreamCtx context.Context, logger zerolog.Logger, haven types.Haven, locator types.Locator, connReq goSrt.ConnRequest) (types.ConnectionSRT, error) {
	logger = logger.With().Str("type", "srt").Logger() // TODO: add IP and location once we have them
	ctx, cancel := context.WithCancel(upstreamCtx)
	c := &connection{
		logger: logger,

		isEscort: false,

		demuxer: multiplexer.NewMpegTsDemuxer(ctx, logger.With().Str("component", "demuxer").Logger(), multiplexer.Settings{
			InputBufferCap:  50,
			OutputBufferCap: 200,
			AudioDriftLimit: 20 * time.Millisecond,
		}),

		haven: haven,

		ctx:    ctx,
		cancel: cancel,
	}

	streamID, err := parseStreamRequest(connReq)
	if err != nil {
		connReq.Reject(goSrt.REJ_ROGUE)
		return nil, err
	}

	if err = validateConnectionRequest(haven, connReq, streamID); err != nil {
		return nil, err
	}

	c.location, err = locator.GetLocation(connReq.RemoteAddr())
	if err != nil {
		c.logger.Error().Err(err).Msg("failed to get location for new connection")
		c.location = types.Location{}
	}
	c.connType = streamID.connectionType

	c.conn, err = connReq.Accept()
	if err != nil {
		connReq.Reject(goSrt.REJ_PEER)
		return nil, fmt.Errorf("failed to accept SRT connection: %w", err)
	}

	// No waitgroup: this goroutine triggers connection close, avoiding deadlock.
	go func() {
		<-c.ctx.Done()
		c.close()
	}()

	return c, nil
}

func (c *connection) GetLocation() types.Location {
	return c.location
}

func (c *connection) GetAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *connection) GetType() types.ConnectionType {
	return c.connType
}

func (c *connection) GetCtx() context.Context {
	return c.ctx
}

func (c *connection) GetLogger() zerolog.Logger {
	return c.logger
}

func (c *connection) WritePacket(pkt packet.Packet) error {
	err := c.conn.WritePacket(pkt)
	if err != nil {
		if errors.Is(err, io.EOF) || strings.Contains(err.Error(), "use of closed network connection") {
			c.logger.Info().Err(err).Msg("SRT connection closed by peer or internally")
			c.Close()
			return err
		}
		c.logger.Error().Err(err).Msg("SRT connection write error")
		c.Close()
		return err
	}

	return nil
}

func (c *connection) Read() chan packet.Packet {
	pktCh, errCh := c.read()
	c.wg.Go(func() {
		for {
			select {
			case <-c.ctx.Done():
				return
			case err := <-errCh:
				if err != nil {
					c.logger.Error().Err(err).Msg("SRT connection read error")
					c.Close()
					return
				}
			}
		}
	})

	return pktCh
}

func (c *connection) Close() {
	c.cancel()
}

func (c *connection) close() {
	err := c.conn.Close()
	if err != nil {
		c.logger.Error().Err(err).Msg("Failed to close SRT connection")
	}
	c.wg.Wait()

	c.logger.Info().Msg("SRT connection closed")
}

func (c *connection) read() (chan packet.Packet, chan error) {
	pktCh := make(chan packet.Packet, 6000)
	errCh := make(chan error, 1)

	c.wg.Go(func() {
		defer close(pktCh)
		defer close(errCh)

		tries := 0
		backoff := initialBackoff

		for {
			if err := c.conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
				select {
				case <-c.ctx.Done():
					return
				case errCh <- fmt.Errorf("failed to set read deadline: %w", err):
				}
				continue
			}

			pkt, err := c.conn.ReadPacket()
			if err != nil {
				var netErr net.Error
				switch {
				case errors.Is(err, io.EOF), strings.Contains(err.Error(), "use of closed network connection"):
					if !c.isEscort {
						c.logger.Info().Err(err).Msg("SRT connection closed by peer or internally")
						c.Close()
						return
					}

					tries++
					c.logger.Warn().Err(err).Int("attempt", tries).Dur("backoff", backoff).Msg("Failed to dial SRT server for escort connection, connection closed. Retrying...")
					select {
					case <-c.ctx.Done():
						return
					case <-time.After(backoff):
						conn, err := goSrt.Dial("srt", c.conn.RemoteAddr().String(), c.config)
						if err != nil {
							c.logger.Error().Err(err).Msg("Failed to re-dial SRT server for escort connection")
							backoff = min(time.Duration(float64(backoff)*backoffMultiplier), maxBackoff)
							continue
						}
						c.conn = conn
						c.logger.Info().Msg("Successfully reconnected to SRT server for escort connection")
						tries = 0
						backoff = initialBackoff
						continue
					}

				case errors.As(err, &netErr) && netErr.Timeout():
					c.logger.Debug().Msgf("SRT read deadline exceeded timeout, retrying...")
					continue

				default:
					select {
					case errCh <- fmt.Errorf("failed to read packet: %w", err):
					default:
					}
					continue
				}
			}
			tries = 0
			backoff = initialBackoff

			select {
			case <-c.ctx.Done():
				return
			case pktCh <- pkt:
			default:
				select {
				case <-c.ctx.Done():
					return
				default:
					c.logger.Warn().Msg("SRT packet channel full, dropping packet")
				}
				continue
			}
		}
	})

	return pktCh, errCh
}
