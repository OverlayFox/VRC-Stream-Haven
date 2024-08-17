package types

type PortMapping struct {
	ExternalPort uint16
	Protocol     string
	InternalPort int
	InternalIP   string
	Enabled      bool
	Description  string
}
