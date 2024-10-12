package types

import (
	"fmt"
	"github.com/OverlayFox/VRC-Stream-Haven/logger"
	"github.com/go-ping/ping"
	geo "github.com/kellydunn/golang-geo"
	"github.com/oschwald/geoip2-golang"
	"github.com/rs/zerolog"
	"net"
)

// Escort holds all information about a Escort running as a part of the Haven.
type Escort struct {
	IpAddress      net.IP  `yaml:"publicIpAddress"`
	RtspEgressPort uint16  `yaml:"rtspEgressPort"`
	Latitude       float64 `yaml:"lat"`
	Longitude      float64 `yaml:"lon"`

	Logger *zerolog.Logger
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

func (e *Escort) checkAvailability() bool {
	pinger, err := ping.NewPinger(e.IpAddress.String())
	if err != nil {
		logger.HavenLogger.Warn().Msgf("Failed to create pinger for %s", e.IpAddress.String())
		return false
	}

	pinger.Count = 2
	pinger.Timeout = 500

	err = pinger.Run()
	if err != nil {
		logger.HavenLogger.Warn().Msgf("Failed to ping %s", e.IpAddress.String())
		return false
	}

	stats := pinger.Statistics()
	if stats.PacketLoss != 0 {
		logger.HavenLogger.Warn().Msgf("Packet loss for %s: %f", e.IpAddress.String(), stats.PacketLoss)
		return false
	}

}
