package types

import "net"

type Haven interface {
	GetStreamID() string
	GetPassphrase() string
	GetPublisher() (Connection, error)

	AddConnection(Connection) error

	GetClosestEscort(Location) Connection
	GetViewer(net.Addr) (Connection, error)

	Close()
}
