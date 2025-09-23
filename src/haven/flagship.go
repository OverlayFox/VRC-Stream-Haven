package haven

import (
	"net"

	"github.com/OverlayFox/VRC-Stream-Haven/src/geo"
	"github.com/OverlayFox/VRC-Stream-Haven/src/types"
)

type Flagship struct {
	types.Connection

	location types.Location
}

func NewFlagship(srtSession types.Connection) (types.Flagship, error) {
	flagship := Flagship{
		Connection: srtSession,
	}

	addr, err := net.ResolveUDPAddr("udp", srtSession.GetIp())
	if err != nil {
		return nil, err
	}

	location, err := geo.GetPublicLocation(addr)
	if err != nil {
		return nil, err
	}
	flagship.location = location

	return &flagship, nil
}

func (f *Flagship) GetLocation() types.Location {
	return f.location
}
