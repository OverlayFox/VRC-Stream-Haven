package types

type PortMapping struct {
	ExternalPort uint16
	Protocol     string
	InternalPort uint16
	InternalIP   string
	Enabled      bool
	Description  string
}
