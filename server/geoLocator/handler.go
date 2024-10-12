package geoLocator

import (
	"github.com/OverlayFox/VRC-Stream-Haven/logger"
	"net"
	"strings"

	"github.com/oschwald/geoip2-golang"
)

var GeoDatabase *geoip2.Reader

func init() {
	GeoDatabase, err := geoip2.Open("./geoDatabase/GeoLite2-City.mmdb")
	if err != nil {
		logger.HavenLogger.Fatal().Err(err).Msg("Failed to load GeoLite2-City Database")
	}
}

// LocateIp takes in an ipAddress, with or without port number, and converts it to the Latitude and Longitude.
// It may return an 0,0 if no country was found for the specified IP Address.
func LocateIp(ipAddress string) (*geoip2.City, error) {
	ip, _, err := net.SplitHostPort(ipAddress)
	if err != nil {
		if strings.Contains(err.Error(), "missing port in address") {
			ip = ipAddress
		} else {
			return &geoip2.City{}, err
		}
	}

	foundCity, err := GeoDatabase.City(net.ParseIP(ip))
	if err != nil {
		return &geoip2.City{}, err
	}

	return foundCity, nil
}
