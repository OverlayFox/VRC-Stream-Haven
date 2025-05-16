package types

import (
	"net"
)

type Haven interface {
	GetStreamId() string

	GetFlagship() Flagship
	AddFlagship(MediaSession) error

	AddEscort(MediaSession) error

	GetClosestEscort(net.Addr) (Escort, error)

	GetPassphrase() string

	GetPacketBuffer() (PacketBuffer, error)
}
