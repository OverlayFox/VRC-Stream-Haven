package types

import (
	"net"
)

type Haven interface {
	GetStreamId() string
	GetPassphrase() string

	GetFlagship() Flagship
	AddFlagship(Connection) error
	TooManyEscorts() bool

	AddEscort(Connection) error

	GetClosestEscort(net.Addr) (Escort, error)

	GetPacketBuffer() (PacketBuffer, error)
}
