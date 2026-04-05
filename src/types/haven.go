package types

import (
	"net"
)

type Haven interface {
	GetStreamId() string
	GetPassphrase() string
	GetPublisher() (Connection, error)

	AddConnection(Connection) error

	TooManyEscorts() bool
	GetClosestEscort(net.Addr) (Connection, error)

	Close()
}
