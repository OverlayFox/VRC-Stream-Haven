package types

type Haven interface {
	GetStreamID() string
	GetPassphrase() string
	GetPublisher() (Connection, error)

	AddConnection(Connection) error

	GetClosestEscort(Location) (Connection, error)

	Close()
}
