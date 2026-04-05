package types

import (
	"net"
)

type Haven interface {
	GetStreamID() string
	GetPassphrase() string
	GetPublisher() (Connection, error)

	AddConnection(Connection) error

	TooManyEscorts() bool
	GetClosestEscort(net.Addr) (Connection, error)

	Close()
}
