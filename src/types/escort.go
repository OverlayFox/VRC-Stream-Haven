package types

import "net"

type Escort interface {
	MediaSession
	GetLocation() Location
	GetAddr() net.Addr
}
