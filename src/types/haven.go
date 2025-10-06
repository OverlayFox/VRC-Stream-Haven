package types

import (
	"context"
	"net"
)

type Ship interface {
	GetIp() net.Addr
	GetLocation() Location

	GetMaxAmountOfViewers() int
	SignalClose()
	GetCtx() context.Context
}

type Haven interface {
	GetStreamId() string
	GetPassphrase() string

	GetFlagship() (Ship, error)
	AddFlagship(Connection) error

	AddEscort(Connection) error
	TooManyEscorts() bool

	GetClosestEscort(net.Addr) (Ship, error)

	Close()
}
