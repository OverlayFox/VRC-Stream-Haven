package types

import "net"

type Haven interface {
	GetStreamID() string
	GetPassphrase() string
	GetPublisher() (ConnectionSRT, error)

	AddConnection(Connection) error

	GetClosestEscort(Location) ConnectionSRT
	GetViewer(net.Addr) (ConnectionRTSP, error)

	Close()
}
