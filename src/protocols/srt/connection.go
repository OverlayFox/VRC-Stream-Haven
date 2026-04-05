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

	"github.com/OverlayFox/VRC-Stream-Haven/src/geo"
	"github.com/OverlayFox/VRC-Stream-Haven/src/types"
	goSrt "github.com/datarhei/gosrt"
	"github.com/datarhei/gosrt/packet"
	"github.com/rs/zerolog"
)

type connection struct {
	logger   zerolog.Logger
	conn     goSrt.Conn
	connType types.ConnectionType

	haven    types.Haven
	location types.Location

	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

func NewConnection(logger zerolog.Logger, serverCtx context.Context, haven types.Haven, connReq goSrt.ConnRequest) (types.Connection, error) {
	ctx, cancel := context.WithCancel(serverCtx)
	c := &connection{
		logger: logger,

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

	c.location, err = geo.GetPublicLocation(connReq.RemoteAddr())
	if err != nil {
		connReq.Reject(goSrt.REJ_IPE)
		return nil, err
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

func (c *connection) Write(pkt packet.Packet) {
	err := c.conn.WritePacket(pkt)
	if err != nil {
		if errors.Is(err, io.EOF) || strings.Contains(err.Error(), "use of closed network connection") {
			c.logger.Info().Err(err).Msg("srt connection closed by peer or internally")
			c.Close()
			return
		}
		c.logger.Error().Err(err).Msg("srt connection write error")
		c.Close()
		return
	}
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
					c.logger.Error().Err(err).Msg("srt connection read error")
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
	c.conn.Close()
	c.wg.Wait()

	c.logger.Info().Msg("srt connection closed")
}

func (c *connection) read() (chan packet.Packet, chan error) {
	pktCh := make(chan packet.Packet, 6000)
	errCh := make(chan error, 1)

	c.wg.Go(func() {
		defer close(pktCh)
		defer close(errCh)

		for {
			select {
			case <-c.ctx.Done():
				return
			default:
			}

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
					c.logger.Info().Err(err).Msg("srt connection closed by peer or internally")
					c.Close()
					return
				case errors.As(err, &netErr) && netErr.Timeout():
					c.logger.Debug().Msg("srt read deadline exceeded (timeout), retrying...")
					continue
				default:
					select {
					case errCh <- fmt.Errorf("failed to read packet: %w", err):
					default:
					}
					c.Close()
					return
				}
			}

			if pkt.Header().IsControlPacket {
				continue
			}

			select {
			case <-c.ctx.Done():
				return
			case pktCh <- pkt:
			default:
				select {
				case <-c.ctx.Done():
					return
				case errCh <- errors.New("packet read channel is full, dropping packets"):
				default:
				}
				continue
			}
		}
	})

	return pktCh, errCh
}

//
// Helper functions
//

func validateConnectionRequest(haven types.Haven, req goSrt.ConnRequest, streamID streamRequest) error {
	if req.Version() != 5 {
		req.Reject(goSrt.REJ_VERSION)
		return fmt.Errorf("unsupported SRT version '%d'", req.Version())
	}
	if !req.IsEncrypted() {
		req.Reject(goSrt.REJ_UNSECURE)
		return errors.New("connection is not encrypted")
	}
	if err := req.SetPassphrase(haven.GetPassphrase()); err != nil {
		req.Reject(goSrt.REJ_BADSECRET)
		return fmt.Errorf("failed to set passphrase: %w", err)
	}

	switch streamID.connectionType {
	case types.ConnectionTypePublisher:
		if _, err := haven.GetPublisher(); err == nil {
			req.Reject(goSrt.REJ_ROGUE)
			return errors.New("a publisher is already connected")
		}
	case types.ConnectionTypeEscort:
		if haven.TooManyEscorts() {
			req.Reject(goSrt.REJ_ROGUE)
			return errors.New("too many escorts connected")
		}
	default:
		req.Reject(goSrt.REJ_ROGUE)
		return fmt.Errorf("unsupported connection type '%s'", streamID.connectionType.String())
	}

	return nil
}

type streamRequest struct {
	streamID       string
	connectionType types.ConnectionType
}

func parseStreamRequest(req goSrt.ConnRequest) (streamRequest, error) {
	parts := strings.Split(req.StreamId(), ":")
	if len(parts) != 2 {
		return streamRequest{}, errors.New("invalid stream ID format")
	}
	connectionType := types.ConnectionTypeFromString(parts[0])
	if connectionType == types.ConnectionTypeUnknown {
		return streamRequest{}, fmt.Errorf("unknown connection type '%s'", parts[0])
	}

	return streamRequest{
		streamID:       parts[1],
		connectionType: connectionType,
	}, nil
}
