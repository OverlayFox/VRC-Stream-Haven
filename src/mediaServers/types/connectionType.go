package types

import "fmt"

type ConnectionType string

const (
	ConnectionTypeFlagship ConnectionType = "flagship"
	ConnectionTypeEscort   ConnectionType = "escort"
	ConnectionTypeLeech    ConnectionType = "leech" // Not used yet, but reserved for future use
)

func (c ConnectionType) String() string {
	return string(c)
}

func (c ConnectionType) IsFlagship() bool {
	return c == ConnectionTypeFlagship
}

func ConnectionTypeFromString(value string) (ConnectionType, error) {
	switch value {
	case string(ConnectionTypeFlagship):
		return ConnectionTypeFlagship, nil
	case string(ConnectionTypeEscort):
		return ConnectionTypeEscort, nil
	default:
		return "", fmt.Errorf("unknown connection type: '%s'", value)
	}
}
