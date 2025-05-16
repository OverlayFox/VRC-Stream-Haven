package types

import (
	"context"
	"net"
)

type MediaSession interface {
	GetAddr() net.Addr

	WriteToSession()
	ReadFromSession()

	GetCtx() context.Context

	SignalClose()
}
