package types

import (
	"net"

	vJson "github.com/OverlayFox/VRC-Stream-Haven/src/types/json"
)

type Haven interface {
	GetStreamId() string

	GetFlagship() Flagship
	AddFlagship(MediaSession) error

	AddEscort(MediaSession, vJson.Announce) error

	GetClosestEscort(net.Addr) (Escort, error)

	GetPassphrase() string

	GetPacketBuffer() (PacketBuffer, error)
}
