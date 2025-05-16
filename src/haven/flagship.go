package haven

import (
	"github.com/OverlayFox/VRC-Stream-Haven/src/geo"
	"github.com/OverlayFox/VRC-Stream-Haven/src/types"
)

type Flagship struct {
	types.MediaSession

	location types.Location
}

func NewFlagship(srtSession types.MediaSession) (types.Flagship, error) {
	flagship := Flagship{
		MediaSession: srtSession,
	}

	location, err := geo.GetPublicLocation(srtSession.GetAddr())
	if err != nil {
		return nil, err
	}
	flagship.location = location

	return &flagship, nil
}

func (f *Flagship) GetLocation() types.Location {
	return f.location
}
