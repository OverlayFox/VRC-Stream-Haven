package types

import (
	"context"
	"encoding/json"
	"net"

	"github.com/datarhei/gosrt/packet"
)

type ConnectionType string

const (
	ConnectionTypeUnknown   ConnectionType = "unknown"
	ConnectionTypePublisher ConnectionType = "publish"
	ConnectionTypeReader    ConnectionType = "read"
	ConnectionTypeEscort    ConnectionType = "escort"
	ConnectionTypeLeech     ConnectionType = "leech"
)

var connectionTypeStrings = map[string]ConnectionType{
	"unknown": ConnectionTypeUnknown,
	"publish": ConnectionTypePublisher,
	"read":    ConnectionTypeReader,
	"escort":  ConnectionTypeEscort,
	"leech":   ConnectionTypeLeech,
}

func (c ConnectionType) String() string {
	return string(c)
}

func (ct ConnectionType) MarshalJSON() ([]byte, error) {
	return json.Marshal(ct.String())
}

func (ct *ConnectionType) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	*ct = ConnectionTypeFromString(str)
	return nil
}

func ConnectionTypeFromString(s string) ConnectionType {
	if t, ok := connectionTypeStrings[s]; ok {
		return t
	}
	return ConnectionTypeUnknown
}

func (c ConnectionType) IsFlagship() bool {
	return c == ConnectionTypePublisher
}

type Connection interface {
	GetAddr() net.Addr
	GetType() ConnectionType
	GetCtx() context.Context
	GetLocation() Location

	Write(packet.Packet)
	Read() chan packet.Packet

	Close()
}
