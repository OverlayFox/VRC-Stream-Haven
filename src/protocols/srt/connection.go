package srt

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/OverlayFox/VRC-Stream-Haven/src/types"
	goSrt "github.com/datarhei/gosrt"
	"github.com/rs/zerolog"
)

type streamRequest struct {
	StreamId       string
	ConnectionType types.ConnectionType
}

type ConnectionConfig struct {
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type Connection struct {
	// Core Connection fields
	logger         zerolog.Logger
	conn           goSrt.Conn
	streamId       string
	connectionType types.ConnectionType
	config         ConnectionConfig

	// Media processing
	// demuxer *multiplexer.TsDemuxer
	// muxer   *multiplexer.TsMuxer

	// Lifecycle management
	ctx         context.Context
	cancel      context.CancelFunc
	signalClose chan struct{}
	wg          sync.WaitGroup
}

// NewConnection creates a new SRT connection based on the provided request and stream manager.
// It disconnects if the request is invalid or if the stream is not found.
func NewConnection(logger zerolog.Logger, req goSrt.ConnRequest, haven types.Haven, config ConnectionConfig) (types.Connection, error) {
	err := validateRequest(req)
	if err != nil {
		return nil, fmt.Errorf("invalid SRT connection request: %w", err)
	}

	streamReq, err := parseStreamRequest(req.StreamId())
	if err != nil {
		req.Reject(goSrt.REJ_ROGUE)
		return nil, fmt.Errorf("failed to parse streamIdRequest '%s': %w", req.StreamId(), err)
	}

	if err = validateStreamAccess(logger, haven, streamReq, req); err != nil {
		return nil, fmt.Errorf("failed to validate stream access for '%s': %w", streamReq.StreamId, err)
	}

	conn, err := req.Accept()
	if err != nil {
		req.Reject(goSrt.REJ_ROGUE)
		return nil, fmt.Errorf("failed to accept SRT connection for stream '%s': %w", streamReq.StreamId, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	c := &Connection{
		logger:         logger.With().Str("stream_id", streamReq.StreamId).Logger(),
		conn:           conn,
		streamId:       streamReq.StreamId,
		connectionType: streamReq.ConnectionType,
		config:         config,

		ctx:         ctx,
		cancel:      cancel,
		signalClose: make(chan struct{}, 1),
	}

	// No WaitGroup for monitorClose, due to it being the entrance point for the close signal.
	go c.monitorClose()

	return c, nil
}

func (c *Connection) GetCtx() context.Context {
	return c.ctx
}

func (c *Connection) GetIp() string {
	return c.conn.RemoteAddr().String()
}

func (c *Connection) GetType() types.ConnectionType {
	return c.connectionType
}

// TODO: Implement
func (c *Connection) Read() chan types.Frame {
	return nil
}

func (c *Connection) Write() error {
	return nil
}

func (c *Connection) SignalClose() {
	select {
	case c.signalClose <- struct{}{}:
	default:
	}
}

func (c *Connection) monitorClose() {
	select {
	case <-c.signalClose:
		c.close()
		return
	case <-c.ctx.Done():
		return
	}
}

func (c *Connection) close() {
	c.logger.Debug().Msg("closing connection")

	if c.conn != nil {
		c.conn.Close()
	}

	c.cancel()
	c.wg.Wait()

	// if c.demuxer != nil {
	// 	c.demuxer.Close()
	// }
	// if c.muxer != nil {
	// 	c.muxer.Close()
	// }
}

//
// Helper functions
//

func validateRequest(req goSrt.ConnRequest) error {
	if req.Version() != 5 {
		req.Reject(goSrt.REJ_VERSION)
		return fmt.Errorf("unsupported SRT version: %d", req.Version())
	}

	if !req.IsEncrypted() {
		req.Reject(goSrt.REJ_UNSECURE)
		return fmt.Errorf("clients provided stream is not encrypted")
	}

	return nil
}

func parseStreamRequest(streamIdRequest string) (streamRequest, error) {
	parts := strings.SplitN(streamIdRequest, ":", 2)
	if len(parts) != 2 {
		return streamRequest{}, fmt.Errorf("invalid streamIdRequest format: %s", streamIdRequest)
	}

	connectionType := types.ConnectionTypeFromString(parts[0])
	if connectionType == types.ConnectionTypeUnknown {
		return streamRequest{}, fmt.Errorf("invalid connection type requested: %s", parts[0])
	}

	return streamRequest{
		StreamId:       parts[1],
		ConnectionType: connectionType,
	}, nil
}

func validateStreamAccess(logger zerolog.Logger, haven types.Haven, streamReq streamRequest, req goSrt.ConnRequest) error {
	if haven.GetStreamId() != streamReq.StreamId {
		req.Reject(goSrt.REJ_ROGUE)
		return fmt.Errorf("stream '%s' not found: %w", streamReq.StreamId, errors.New("stream not found"))
	}

	flagship := haven.GetFlagship()
	switch streamReq.ConnectionType {
	case types.ConnectionTypeFlagship:
		logger.Debug().Msg("srt connection requested a flagship request for application")

		if flagship != nil {
			req.Reject(goSrt.REJ_ROGUE)
			return fmt.Errorf("stream '%s' already has a flagship", streamReq.StreamId)
		}
		if err := req.SetPassphrase(haven.GetPassphrase()); err == nil {
			req.Reject(goSrt.REJ_BADSECRET)
			return fmt.Errorf("failed to set passphrase for stream '%s': %w", streamReq.StreamId, err)
		}
		logger.Debug().Msg("allowing srt connection to start publishing to application")

	case types.ConnectionTypeEscort:
		logger.Debug().Msg("srt connection requested a play request for application")

		if flagship == nil {
			req.Reject(goSrt.REJ_ROGUE)
			return fmt.Errorf("stream '%s' does not have an active flagship yet", streamReq.StreamId)
		}
		if err := req.SetPassphrase(haven.GetPassphrase()); err == nil {
			req.Reject(goSrt.REJ_BADSECRET)
			return fmt.Errorf("failed to set passphrase for stream '%s': %w", streamReq.StreamId, err)
		}

		if haven.TooManyEscorts() {
			req.Reject(goSrt.REJ_RESOURCE)
			return fmt.Errorf("too many consumers for stream '%s'", streamReq.StreamId)
		}
		if _, err := haven.GetPacketBuffer(); err != nil {
			req.Reject(goSrt.REJ_RESOURCE)
			return fmt.Errorf("stream '%s' buffer not ready", streamReq.StreamId)
		}
		logger.Debug().Msg("allowing srt connection to start reading from application")

	default:
		req.Reject(goSrt.REJ_ROGUE)
		return fmt.Errorf("unknown connection type '%s'", streamReq.ConnectionType)
	}

	return nil
}
