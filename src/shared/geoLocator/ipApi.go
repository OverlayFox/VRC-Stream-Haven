package geoLocator

import (
	"encoding/json"
	"fmt"
	"net"
)

type PublicLocation struct {
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lon"`
	IpAddress net.IP  `json:"query"`
}

func (ip *PublicLocation) UnmarshalJSON(data []byte) error {
	type Alias PublicLocation
	aux := &struct {
		IpAddress string `json:"query"`
		*Alias
	}{
		Alias: (*Alias)(ip),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	ip.IpAddress = net.ParseIP(aux.IpAddress)
	if ip.IpAddress == nil {
		return fmt.Errorf("invalid IP address: %s", aux.IpAddress)
	}

	return nil
}
