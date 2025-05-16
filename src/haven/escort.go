package haven

import (
	"github.com/OverlayFox/VRC-Stream-Haven/src/geo"
	"github.com/OverlayFox/VRC-Stream-Haven/src/types"
)

type Escort struct {
	types.MediaSession

	location types.Location
}

func NewEscort(srtSession types.MediaSession) (types.Escort, error) {
	escort := &Escort{
		MediaSession: srtSession,
	}

	location, err := geo.GetPublicLocation(srtSession.GetAddr())
	if err != nil {
		return nil, err
	}
	escort.location = location

	return escort, nil
}

func (e *Escort) GetLocation() types.Location {
	return e.location
}
