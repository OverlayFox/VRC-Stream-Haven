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
	streamId   string
	passphrase string

	flagship types.Flagship
	escorts  sync.Map

	packetBuffer types.PacketBuffer

	logger zerolog.Logger
}

var _ types.Haven = (*Haven)(nil)

func NewHaven(passphrase, streamId string, packetBuffer types.PacketBuffer, logger zerolog.Logger) types.Haven {
	return &Haven{
		streamId:   streamId,
		passphrase: passphrase,

		flagship:     nil,
		escorts:      sync.Map{},
		packetBuffer: packetBuffer,

		logger: logger,
	}
}

func (h *Haven) GetStreamId() string {
	return h.streamId
}

func (h *Haven) GetFlagship() types.Flagship {
	return h.flagship
}

func (h *Haven) AddFlagship(mediaSession types.MediaSession) error {
	if h.flagship != nil {
		return types.ErrFlagshipAlreadyExists
	}

	flagship, err := NewFlagship(mediaSession)
	if err != nil {
		return err
	}
	h.flagship = flagship

	// Monitor the flagship's context and remove it from the map when it is done
	// Closes the escorts as well
	go func(ctx context.Context) {
		<-ctx.Done()
		h.logger.Debug().Msg("Flagship context done, closing all escorts")
		h.escorts.Range(func(_, value any) bool {
			escort := value.(types.Escort)
			escort.SignalClose()
			return true
		})
		h.flagship = nil
	}(flagship.GetCtx())

	return nil
}

func (h *Haven) AddEscort(mediaSession types.MediaSession) error {
	if h.flagship == nil {
		return types.ErrFlagshipNotFound
	}

	escort, err := NewEscort(mediaSession)
	if err != nil {
		return err
	}
	h.escorts.Store(escort.GetAddr().String(), escort)

	// Monitor the escort's context and remove it from the map when it is done
	go func(ctx context.Context, key string) {
		<-ctx.Done()
		h.logger.Debug().Msgf("Escort context done, removing escort '%s' from map", key)
		h.escorts.Delete(key)
	}(escort.GetCtx(), escort.GetAddr().String())

	return nil
}

func (h *Haven) GetClosestEscort(clientAddr net.Addr) (types.Escort, error) {
	empty := true
	h.escorts.Range(func(_, _ any) bool {
		empty = false
		return false
	})
	if empty {
		return nil, types.ErrEscortsNotAvailable
	}

	clientLocation, err := geo.GetPublicLocation(clientAddr)
	if err != nil {
		return nil, err
	}

	var closestEscort types.Escort
	clientGeoPoint := clientLocation.GetGeoPoint()
	flagshipLocation := h.flagship.GetLocation()
	closestDistance := flagshipLocation.GetDistanceBetween(clientGeoPoint)

	h.escorts.Range(func(_, value any) bool {
		escort := value.(types.Escort)
		escortLocation := escort.GetLocation()
		distance := escortLocation.GetDistanceBetween(clientGeoPoint)

		if distance < closestDistance {
			closestDistance = distance
			closestEscort = escort
		}
		return true
	})

	return closestEscort, nil
}

func (h *Haven) GetPassphrase() string {
	return h.passphrase
}

func (h *Haven) GetPacketBuffer() (types.PacketBuffer, error) {
	if h.packetBuffer == nil {
		return nil, types.ErrBufferNotReady
	}

	return h.packetBuffer, nil
}

func (h *Haven) Close() {
	h.logger.Debug().Msg("Closing Haven")

	if h.flagship != nil {
		h.flagship.SignalClose()
	}

	h.escorts.Range(func(_, value any) bool {
		escort := value.(types.Escort)
		escort.SignalClose()
		return true
	})

	if h.packetBuffer == nil {
		h.packetBuffer.Close()
		h.packetBuffer = nil
	}

}
