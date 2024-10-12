package types

import (
	"fmt"
	geo "github.com/kellydunn/golang-geo"
	"github.com/oschwald/geoip2-golang"
	"net"
)

// Escort holds all information about a Escort running as a part of the Haven.
type Escort struct {
	IpAddress      net.IP  `yaml:"publicIpAddress"`
	RtspEgressPort uint16  `yaml:"rtspEgressPort"`
	Latitude       float64 `yaml:"lat"`
	Longitude      float64 `yaml:"lon"`
	Username       string  `yaml:"username"`
	Passphrase     string  `yaml:"passphrase"`
}

func (e *Escort) getPoint() *geo.Point {
	return geo.NewPoint(e.Latitude, e.Longitude)
}

// GetDistance returns the distance between an escort and another client.
// It returns the distance in kilometres.
func (e *Escort) GetDistance(city *geoip2.City) (float64, error) {
	if city.Location.Latitude == 0 && city.Location.Longitude == 0 {
		return 0, fmt.Errorf("city does not have a valid latitude or longitude")
	}
	clientLocation := geo.NewPoint(city.Location.Latitude, city.Location.Longitude)

	return clientLocation.GreatCircleDistance(e.getPoint()), nil
}
