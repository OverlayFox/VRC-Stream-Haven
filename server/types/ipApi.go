package types

import (
	"encoding/json"
	"fmt"
	"net"
)

type IpApi struct {
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lon"`
	IpAddress net.IP  `json:"query"`
}

func (ip *IpApi) UnmarshalJSON(data []byte) error {
	type Alias IpApi
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
