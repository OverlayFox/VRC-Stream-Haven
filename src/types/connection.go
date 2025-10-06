package types

import (
	"context"
	"encoding/json"
	"net"
)

// ConnectionType represents the type of connection (publisher, subscriber, etc)
type ConnectionType string

const (
	ConnectionTypeUnknown  ConnectionType = "unknown"
	ConnectionTypeFlagship ConnectionType = "flagship"
	ConnectionTypeEscort   ConnectionType = "escort"
	ConnectionTypeLeech    ConnectionType = "leech" // Not used yet, but reserved for future use
)

var connectionTypeStrings = map[string]ConnectionType{
	"flagship": ConnectionTypeFlagship,
	"escort":   ConnectionTypeEscort,
	"leech":    ConnectionTypeLeech,
}

// String implements the Stringer interface
func (c ConnectionType) String() string {
	return string(c)
}

// MarshalJSON implements the json.Marshaler interface
func (ct ConnectionType) MarshalJSON() ([]byte, error) {
	return json.Marshal(ct.String())
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (ct *ConnectionType) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	if t, ok := connectionTypeStrings[str]; ok {
		*ct = t
		return nil
	}
	*ct = ConnectionTypeUnknown
	return nil
}

// ConnectionTypeFromString converts a string to ConnectionType
func ConnectionTypeFromString(s string) ConnectionType {
	if t, ok := connectionTypeStrings[s]; ok {
		return t
	}
	return ConnectionTypeUnknown
}

func (c ConnectionType) IsFlagship() bool {
	return c == ConnectionTypeFlagship
}

type Connection interface {
	GetAddr() net.Addr
	GetType() ConnectionType
	GetCtx() context.Context

	Write() error
	Read() chan Frame

	SignalClose()
}
