package haven

import (
	"context"
	"net"
	"sync"

	"github.com/OverlayFox/VRC-Stream-Haven/src/geo"
	"github.com/OverlayFox/VRC-Stream-Haven/src/types"
	"github.com/rs/zerolog"
)

type Haven struct {
	logger zerolog.Logger

	streamId   string
	passphrase string

	flagship types.Ship   // flagship provides the main stream
	escorts  []types.Ship // escorts are nodes that relay the flagship's stream to viewers
	leeches  []types.Ship // leeches are nodes that only consume the stream and don't relay it

	flagshipMtx sync.RWMutex
	escortMtx   sync.RWMutex
	leechMtx    sync.RWMutex

	buffer types.PacketBuffer

	ctx    context.Context
	cancel context.CancelFunc
}

var _ types.Haven = (*Haven)(nil)

func NewHaven(passphrase, streamId string, packetBuffer types.PacketBuffer, logger zerolog.Logger) types.Haven {
	ctx, cancel := context.WithCancel(context.Background())
	return &Haven{
		logger: logger,

		streamId:   streamId,
		passphrase: passphrase,

		flagship: nil,
		escorts:  make([]types.Ship, 0),
		leeches:  make([]types.Ship, 0),

		buffer: packetBuffer,

		ctx:    ctx,
		cancel: cancel,
	}
}

func (h *Haven) GetStreamId() string {
	return h.streamId
}

func (h *Haven) GetPassphrase() string {
	return h.passphrase
}

func (h *Haven) GetFlagship() (types.Ship, error) {
	h.flagshipMtx.RLock()
	defer h.flagshipMtx.RUnlock()

	if h.flagship == nil {
		return nil, types.ErrFlagshipNotFound
	}
	return h.flagship, nil
}

func (h *Haven) AddFlagship(mediaSession types.Connection) error {
	h.flagshipMtx.Lock()
	defer h.flagshipMtx.Unlock()

	if h.flagship != nil {
		return types.ErrFlagshipAlreadyExists
	}

	flagship, err := NewShip(mediaSession)
	if err != nil {
		return err
	}
	h.flagship = flagship

	// Monitor the flagship's context and remove it from the map when it is done
	// Closes the escorts as well
	go func() {
		select {
		case <-h.ctx.Done():
			h.logger.Debug().Msg("Haven context done, closing flagship")

			h.flagship.SignalClose()

		case <-flagship.GetCtx().Done():
			h.logger.Debug().Msg("Flagship context done, closing flagship")

			h.escortMtx.Lock()
			defer h.escortMtx.Unlock()
			for _, escort := range h.escorts {
				h.logger.Debug().Msgf("Closing escort '%s' as flagship has disconnected", escort.GetIp().String())
				escort.SignalClose()
			}
			h.escorts = make([]types.Ship, 0)

			h.leechMtx.Lock()
			defer h.leechMtx.Unlock()
			for _, leech := range h.leeches {
				h.logger.Debug().Msgf("Closing leech '%s' as flagship has disconnected", leech.GetIp().String())
				leech.SignalClose()
			}
			h.leeches = make([]types.Ship, 0)

			h.flagshipMtx.Lock()
			defer h.flagshipMtx.Unlock()
			h.flagship = nil
		}
	}()

	return nil
}

func (h *Haven) AddEscort(mediaSession types.Connection) error {
	h.flagshipMtx.RLock()
	if h.flagship == nil {
		h.flagshipMtx.RUnlock()
		return types.ErrFlagshipNotFound
	}
	h.flagshipMtx.RUnlock()

	escort, err := NewShip(mediaSession)
	if err != nil {
		return err
	}
	h.escorts = append(h.escorts, escort)

	// Monitor the escort's context and remove it from the map when it is done
	go func() {
		select {
		case <-h.ctx.Done():
			h.logger.Debug().Msgf("Haven context done, closing escort '%s'", escort.GetIp().String())
			escort.SignalClose()

		case <-escort.GetCtx().Done():
			h.logger.Debug().Msgf("Escort context done, closing escort '%s'", escort.GetIp().String())

			h.escortMtx.Lock()
			defer h.escortMtx.Unlock()
			for i, e := range h.escorts {
				if e == escort {
					h.escorts = append(h.escorts[:i], h.escorts[i+1:]...)
					break
				}
			}
		}
	}()

	return nil
}

func (h *Haven) TooManyEscorts() bool {
	h.escortMtx.RLock()
	defer h.escortMtx.RUnlock()

	return len(h.escorts) >= 10
}

func (h *Haven) AddLeech(mediaSession types.Connection) error {
	h.flagshipMtx.RLock()
	if h.flagship == nil {
		h.flagshipMtx.RUnlock()
		return types.ErrFlagshipNotFound
	}
	h.flagshipMtx.RUnlock()

	h.leechMtx.Lock()
	defer h.leechMtx.Unlock()

	leech, err := NewShip(mediaSession)
	if err != nil {
		return err
	}
	h.leeches = append(h.leeches, leech)

	// Monitor the leech's context and remove it from the map when it is done
	go func() {
		select {
		case <-h.ctx.Done():
			h.logger.Debug().Msgf("Haven context done, closing leech '%s'", leech.GetIp().String())
			leech.SignalClose()

		case <-leech.GetCtx().Done():
			h.logger.Debug().Msgf("Leech context done, closing leech '%s'", leech.GetIp().String())

			for i, l := range h.leeches {
				if l == leech {
					h.leeches = append(h.leeches[:i], h.leeches[i+1:]...)
					break
				}
			}
		}
	}()

	return nil
}

func (h *Haven) GetClosestEscort(clientAddr net.Addr) (types.Ship, error) {
	if len(h.escorts) <= 0 {
		return nil, types.ErrEscortsNotAvailable
	}

	clientLocation, err := geo.GetPublicLocation(clientAddr)
	if err != nil {
		return nil, err
	}

	var closestEscort types.Ship
	clientGeoPoint := clientLocation.GetGeoPoint()
	flagshipLocation := h.flagship.GetLocation()
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
	h.flagshipMtx.Lock()
	defer h.flagshipMtx.Unlock()

	h.escortMtx.Lock()
	defer h.escortMtx.Unlock()

	h.logger.Debug().Msg("Closing Haven")

	if h.flagship != nil {
		h.flagship.SignalClose()
	}

	for _, escort := range h.escorts {
		escort.SignalClose()
	}
	h.escorts = make([]types.Ship, 0)
}
