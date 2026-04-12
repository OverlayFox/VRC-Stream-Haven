package haven

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/OverlayFox/VRC-Stream-Haven/src/buffer"
	"github.com/OverlayFox/VRC-Stream-Haven/src/multiplexer"
	"github.com/OverlayFox/VRC-Stream-Haven/src/types"
	"github.com/datarhei/gosrt/packet"
	"github.com/rs/zerolog"
)

type Haven struct {
	logger zerolog.Logger

	streamID   string
	passphrase string

	publisher types.Connection   // publisher provides the main stream
	escorts   []types.Connection // escorts are nodes that relay the publisher's stream to viewers

	publisherMtx sync.RWMutex
	escortMtx    sync.RWMutex
	viewersMtx   sync.RWMutex

	demuxer *multiplexer.MpegTsDemuxer
	buffer  types.Buffer

	locator types.Locator

	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

var _ types.Haven = (*Haven)(nil)

func NewHaven(upstreamCtx context.Context, logger zerolog.Logger, locator types.Locator, passphrase, streamID string) (types.Haven, error) {
	demuxerConfig := multiplexer.Settings{
		InputBufferCap:  100,
		OutputBufferCap: 200,
		AudioDriftLimit: 20 * time.Millisecond,
	}
	ctx, cancel := context.WithCancel(context.Background())
	demuxer := multiplexer.NewMpegTsDemuxer(ctx, logger, demuxerConfig)

	return &Haven{
		logger: logger,

		streamID:   streamID,
		passphrase: passphrase,

		publisher: nil,
		escorts:   make([]types.Connection, 0),

		demuxer: demuxer,
		buffer:  buffer.NewBuffer(logger),

		locator: locator,

		ctx:    ctx,
		cancel: cancel,
	}, nil
}

func (h *Haven) GetStreamID() string {
	return h.streamID
}

func (h *Haven) GetPassphrase() string {
	return h.passphrase
}

func (h *Haven) GetPublisher() (types.Connection, error) {
	h.publisherMtx.RLock()
	defer h.publisherMtx.RUnlock()

	if h.publisher == nil {
		return nil, types.ErrPublisherNotFound
	}
	return h.publisher, nil
}

func (h *Haven) AddConnection(conn types.Connection) error {
	switch conn.GetType() {
	case types.ConnectionTypeEscort:
		return h.addEscort(conn)
	case types.ConnectionTypePublisher:
		return h.addPublisher(conn)
	default:
		return errors.New("unknown connection type")
	}
}

func (h *Haven) GetClosestEscort(location types.Location) (types.Connection, error) {
	if len(h.escorts) == 0 {
		return nil, types.ErrEscortsNotAvailable
	}

	var closestEscort types.Connection
	clientGeoPoint := location.GetGeoPoint()
	flagshipLocation := h.publisher.GetLocation()
	closestDistance := flagshipLocation.GetDistanceBetween(clientGeoPoint)

	for _, escort := range h.escorts {
		escortLocation := escort.GetLocation()
		distance := escortLocation.GetDistanceBetween(clientGeoPoint)

		if distance < closestDistance {
			closestDistance = distance
			closestEscort = escort
		}
	}

	if closestEscort == nil {
		return nil, types.ErrEscortsNotAvailable
	}

	return closestEscort, nil
}

func (h *Haven) Close() {
	h.logger.Debug().Msg("Closing Haven")

	h.cancel()
	h.wg.Wait()

	h.publisherMtx.Lock()
	defer h.publisherMtx.Unlock()
	if h.publisher != nil {
		h.publisher.Close()
		h.publisher = nil
	}

	h.escortMtx.Lock()
	defer h.escortMtx.Unlock()
	for _, escort := range h.escorts {
		escort.Close()
	}
	h.escorts = make([]types.Connection, 0)

	h.buffer.Close()
	h.demuxer.Close()
}

func (h *Haven) addPublisher(conn types.Connection) error {
	h.publisherMtx.Lock()
	defer h.publisherMtx.Unlock()

	if h.publisher != nil {
		return types.ErrPublisherAlreadyExists
	}
	h.publisher = conn

	// Monitor the flagship's context and remove it from the map when it is done
	// Closes the escorts as well
	h.wg.Go(func() {
		select {
		case <-h.ctx.Done():
			h.logger.Debug().Msg("Haven context done, closing publisher")
			h.publisher.Close()

		case <-h.publisher.GetCtx().Done():
			h.logger.Debug().Msg("Publisher context done, removing publisher from haven")

			h.escortMtx.Lock()
			defer h.escortMtx.Unlock()
			for _, escort := range h.escorts {
				h.logger.Info().Msgf("Closing escort '%s' as publisher has disconnected", escort.GetAddr().String())
				escort.Close()
			}
			h.escorts = make([]types.Connection, 0)

			h.publisherMtx.Lock()
			defer h.publisherMtx.Unlock()
			h.publisher = nil
		}
	})

	// Start reading from the publisher and writing to the buffer and demuxer
	go func() {
		packetCh := h.publisher.Read()
		demuxerPktCh := make(chan packet.Frame, 100)
		frameCh, errCh := h.demuxer.StartDemuxer(demuxerPktCh)

		h.wg.Go(func() {
			for {
				select {
				case <-h.ctx.Done():
					h.logger.Debug().Msg("Haven context done, closing demuxer")
					h.demuxer.Close()
					return
				case err := <-errCh:
					if err != nil {
						h.logger.Error().Err(err).Msg("Haven demuxer error")
					}
					return

				case frame, ok := <-frameCh:
					if !ok {
						h.logger.Debug().Msg("Frame channel closed")
						return
					}
					err := h.buffer.Write(frame.Clone())
					if err != nil {
						h.logger.Error().Err(err).Msg("Haven buffer write error")
					}
					frame.Decommission()
				}
			}
		})

		for {
			select {
			case <-h.ctx.Done():
				return
			case p, ok := <-packetCh:
				if !ok {
					h.logger.Debug().Msg("Packet channel closed")
					return
				}

				demuxerClone := p.Clone()
				select {
				case demuxerPktCh <- demuxerClone:
				default:
					demuxerClone.Decommission()
				}

				for _, escort := range h.escorts {
					escortClone := p.Clone()
					escort.Write(escortClone)
				}
				p.Decommission()
			}
		}
	}()

	return nil
}

func (h *Haven) addEscort(conn types.Connection) error {
	h.publisherMtx.RLock()
	if h.publisher == nil {
		h.publisherMtx.RUnlock()
		return types.ErrPublisherNotFound
	}
	h.publisherMtx.RUnlock()

	h.escorts = append(h.escorts, conn)

	// Monitor the escort's context and remove it from the map when it is done
	h.wg.Go(func() {
		select {
		case <-h.ctx.Done():
			h.logger.Debug().Msgf("Haven context done, closing escort '%s'", conn.GetAddr().String())
			conn.Close()

		case <-conn.GetCtx().Done():
			h.logger.Debug().Msgf("Escort context done, removing escort '%s' from haven", conn.GetAddr().String())

			h.escortMtx.Lock()
			defer h.escortMtx.Unlock()
			for i, e := range h.escorts {
				if e == conn {
					h.escorts = append(h.escorts[:i], h.escorts[i+1:]...)
					break
				}
			}
		}
	})

	return nil
}

func (h *Haven) GetRTSPStream() ([]types.BufferOutput, error) {
	h.logger.Debug().Msg("Received request for RTSP stream from haven")
	h.publisherMtx.RLock()
	if h.publisher == nil {
		h.publisherMtx.RUnlock()
		return nil, types.ErrPublisherNotFound
	}
	h.publisherMtx.RUnlock()

	return h.buffer.Subscribe(h.ctx, -2*time.Second)
}
