package types

type ProtocolServer interface {
	Start()
	Close()
	Dial(address string, streamID, passphrase string) error
}
