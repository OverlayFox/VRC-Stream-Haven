package haven

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/OverlayFox/VRC-Stream-Haven/src/buffer"
	"github.com/OverlayFox/VRC-Stream-Haven/src/geo"
	"github.com/OverlayFox/VRC-Stream-Haven/src/multiplexer"
	"github.com/OverlayFox/VRC-Stream-Haven/src/types"
	"github.com/datarhei/gosrt/packet"
	"github.com/rs/zerolog"
)

type Haven struct {
	logger zerolog.Logger

	streamId   string
	passphrase string

	publisher types.Connection   // publisher provides the main stream
	escorts   []types.Connection // escorts are nodes that relay the publisher's stream to viewers
	leeches   []types.Connection // leeches are nodes that only consume the stream and don't relay it

	PublisherMtx sync.RWMutex
	escortMtx    sync.RWMutex
	leechMtx     sync.RWMutex

	demuxer *multiplexer.MpegTsDemuxer
	buffer  types.Buffer

	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
}

var _ types.Haven = (*Haven)(nil)

func NewHaven(passphrase, streamId string, logger zerolog.Logger) (types.Haven, error) {
	demuxerConfig := multiplexer.Settings{
		InputBufferCap:  100,
		OutputBufferCap: 200,
		AudioDriftLimit: 20 * time.Millisecond,
	}
	demuxer := multiplexer.NewMpegTsDemuxer(logger.With().Str("component", "ts_demuxer").Logger(), demuxerConfig, context.Background())

	ctx, cancel := context.WithCancel(context.Background())
	return &Haven{
		logger: logger,

		streamId:   streamId,
		passphrase: passphrase,

		publisher: nil,
		escorts:   make([]types.Connection, 0),
		leeches:   make([]types.Connection, 0),

		demuxer: demuxer,
		buffer:  buffer.NewBuffer(logger.With().Str("component", fmt.Sprintf("%s_buffer", streamId)).Logger()),

		ctx:    ctx,
		cancel: cancel,
	}, nil
}

func (h *Haven) GetStreamId() string {
	return h.streamId
}

func (h *Haven) GetPassphrase() string {
	return h.passphrase
}

func (h *Haven) GetPublisher() (types.Connection, error) {
	h.PublisherMtx.RLock()
	defer h.PublisherMtx.RUnlock()

	if h.publisher == nil {
		return nil, types.ErrPublisherNotFound
	}
	return h.publisher, nil
}

func (h *Haven) AddConnection(conn types.Connection) error {
	switch conn.GetType() {
	case types.ConnectionTypeEscort:
		return h.addEscort(conn)
	case types.ConnectionTypeLeech:
		return h.addLeech(conn)
	case types.ConnectionTypePublisher:
		return h.addPublisher(conn)
	default:
		return errors.New("unknown connection type")
	}
}

func (h *Haven) addPublisher(conn types.Connection) error {
	h.PublisherMtx.Lock()
	defer h.PublisherMtx.Unlock()

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

			h.leechMtx.Lock()
			defer h.leechMtx.Unlock()
			for _, leech := range h.leeches {
				h.logger.Info().Msgf("Closing leech '%s' as publisher has disconnected", leech.GetAddr().String())
				leech.Close()
			}
			h.leeches = make([]types.Connection, 0)

			h.PublisherMtx.Lock()
			defer h.PublisherMtx.Unlock()
			h.publisher = nil
		}
	})

	go func() {
		packetCh := h.publisher.Read()
		demuxerPktCh := make(chan packet.Packet, 100)
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
					h.buffer.Write(frame.Clone())
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
	h.PublisherMtx.RLock()
	if h.publisher == nil {
		h.PublisherMtx.RUnlock()
		return types.ErrPublisherNotFound
	}
	h.PublisherMtx.RUnlock()

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

func (h *Haven) TooManyEscorts() bool {
	h.escortMtx.RLock()
	defer h.escortMtx.RUnlock()

	// TODO: Make this configurable
	return len(h.escorts) >= 10
}

func (h *Haven) addLeech(conn types.Connection) error {
	h.PublisherMtx.RLock()
	if h.publisher == nil {
		h.PublisherMtx.RUnlock()
		return types.ErrPublisherNotFound
	}
	h.PublisherMtx.RUnlock()

	h.leechMtx.Lock()
	defer h.leechMtx.Unlock()
	h.leeches = append(h.leeches, conn)

	// Monitor the leech's context and remove it from the map when it is done
	h.wg.Go(func() {
		select {
		case <-h.ctx.Done():
			h.logger.Debug().Msgf("Haven context done, closing leech '%s'", conn.GetAddr().String())
			conn.Close()

		case <-conn.GetCtx().Done():
			h.logger.Debug().Msgf("Leech context done, removing leech '%s' from haven", conn.GetAddr().String())

			for i, l := range h.leeches {
				if l == conn {
					h.leeches = append(h.leeches[:i], h.leeches[i+1:]...)
					break
				}
			}
		}
	})

	return nil
}

func (h *Haven) GetClosestEscort(clientAddr net.Addr) (types.Connection, error) {
	if len(h.escorts) <= 0 {
		return nil, types.ErrEscortsNotAvailable
	}

	clientLocation, err := geo.GetPublicLocation(clientAddr)
	if err != nil {
		return nil, err
	}

	var closestEscort types.Connection
	clientGeoPoint := clientLocation.GetGeoPoint()
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

	h.PublisherMtx.Lock()
	defer h.PublisherMtx.Unlock()

	h.escortMtx.Lock()
	defer h.escortMtx.Unlock()

	h.leechMtx.Lock()
	defer h.leechMtx.Unlock()
}
