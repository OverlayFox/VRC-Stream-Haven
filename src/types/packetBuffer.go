package types

import "github.com/datarhei/gosrt/packet"

type PacketBuffer interface {
	Subscribe() chan packet.Packet
	Unsubscribe(chan packet.Packet)
	Write(packet.Packet)
	Close()
}
