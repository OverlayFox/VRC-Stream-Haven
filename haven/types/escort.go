package types

import "net"

// Escort holds all information about a Escort running as a part of the Haven.
type Escort struct {
	IpAddress      net.IP  `yaml:"publicIpAddress"`
	RtspEgressPort int     `yaml:"rtspEgressPort"`
	Latitude       float64 `yaml:"lat"`
	Longitude      float64 `yaml:"lon"`
}
