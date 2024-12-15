package overseer

import (
	"encoding/json"
	"fmt"
	geo "github.com/kellydunn/golang-geo"
	"github.com/oschwald/geoip2-golang"
	"net"
)

// Escort holds all information about an Escort running as a part of the Haven.
type Escort struct {
	IpAddress      net.IP  `yaml:"publicIpAddress"`
	BackEndIP      net.IP  `yaml:"backEndIP"`
	RtspEgressPort uint16  `yaml:"rtspEgressPort"`
	ApiPort        uint16  `yaml:"apiPort"`
	Latitude       float64 `yaml:"lat"`
	Longitude      float64 `yaml:"lon"`
}

// MakeEscort creates an Escort
func MakeEscort(rtspEgressPort, apiPort uint16, rtspIp, backendIp net.IP) *Escort {
	return &Escort{
		IpAddress:      rtspIp,
		BackEndIP:      backendIp,
		RtspEgressPort: rtspEgressPort,
		Latitude:       rtspIp.Latitude,
		Longitude:      rtspIp.Longitude,
		ApiPort:        apiPort,
	}
}

func (e *Escort) MarshalJSON() ([]byte, error) {
	type Alias Escort
	return json.Marshal(&struct {
		IpAddress string `json:"publicIpAddress"`
		BackEndIP string `json:"backEndIP"`
		*Alias
	}{
		IpAddress: e.IpAddress.String(),
		BackEndIP: e.BackEndIP.String(),
		Alias:     (*Alias)(e),
	})
}

func (e *Escort) UnmarshalJSON(data []byte) error {
	type Alias Escort
	aux := &struct {
		IpAddress string `json:"publicIpAddress"`
		BackEndIP string `json:"backEndIP"`
		*Alias
	}{
		Alias: (*Alias)(e),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	e.IpAddress = net.ParseIP(aux.IpAddress)
	if e.IpAddress == nil {
		return fmt.Errorf("invalid IP address: %s", aux.IpAddress)
	}

	e.BackEndIP = net.ParseIP(aux.BackEndIP)
	if e.BackEndIP == nil {
		return fmt.Errorf("invalid back-end IP address: %s", aux.BackEndIP)
	}

	return nil
}

func (e *Escort) getGeoPoint() *geo.Point {
	return geo.NewPoint(e.Latitude, e.Longitude)
}

// GetDistance returns the distance between an escort and another client.
// It returns the distance in kilometres.
func (e *Escort) GetDistance(city *geoip2.City) (float64, error) {
	if city.Location.Latitude == 0 && city.Location.Longitude == 0 {
		return 0, fmt.Errorf("city does not have a valid latitude or longitude")
	}
	clientLocation := geo.NewPoint(city.Location.Latitude, city.Location.Longitude)

	return clientLocation.GreatCircleDistance(e.getGeoPoint()), nil
}
