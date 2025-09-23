package haven

import (
	"net"

	"github.com/OverlayFox/VRC-Stream-Haven/src/geo"
	"github.com/OverlayFox/VRC-Stream-Haven/src/types"
)

type Escort struct {
	types.Connection

	location types.Location
}

func NewEscort(srtSession types.Connection) (types.Escort, error) {
	escort := &Escort{
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
	escort.location = location

	return escort, nil
}

func (e *Escort) GetLocation() types.Location {
	return e.location
}
