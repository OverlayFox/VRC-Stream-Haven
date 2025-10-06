package haven

import (
	"log"
	"net"

	"github.com/OverlayFox/VRC-Stream-Haven/src/geo"
	"github.com/OverlayFox/VRC-Stream-Haven/src/types"
)

type Ship struct {
	types.Connection

	location types.Location

	consumers []types.Connection // rtsp connections that consume the ships stream
	buffer    types.PacketBuffer
}

func NewShip(conn types.Connection) (types.Ship, error) {
	s := Ship{
		Connection: conn,
	}

	location, err := geo.GetPublicLocation(s.GetAddr())
	if err != nil {
		return nil, err
	}
	s.location = location

	return &s, nil
}

func (s *Ship) GetIp() (addr net.Addr) {
	return s.Connection.GetAddr()
}

func (s *Ship) GetLocation() types.Location {
	return s.location
}

func (s *Ship) GetMaxAmountOfViewers() int {
	log.Panic("Implement me")
	return 0
}
