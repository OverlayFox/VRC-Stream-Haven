package types

type RtspProtocolServer interface {
	Start()
	Close()
}

type SrtProtocolServer interface {
	Start()
	Close()
	Call(address string) error
}
