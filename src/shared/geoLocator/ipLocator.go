package geoLocator

import (
	"encoding/json"
	"github.com/OverlayFox/VRC-Stream-Haven/shared/logger"
	"net"
	"net/http"
	"strings"

	"github.com/oschwald/geoip2-golang"
)

var GeoDatabase *geoip2.Reader

// GetIpLocation takes in an ipAddress, with or without port number, and converts it to the Latitude and Longitude.
// It may return an 0,0 if no country was found for the specified IP Address.
func GetIpLocation(ipAddress string) (*geoip2.City, error) {
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

// GetCurrentPublicLocation Gets the Public IP Address of the Server running this function.
func GetCurrentPublicLocation() (PublicLocation, error) {
	response, err := http.Get("http://ip-api.com/json")
	if err != nil {
		logger.HavenLogger.Error().Err(err).Msg("IP-API is not reachable.")
		return PublicLocation{}, err
	}

	var body PublicLocation
	err = json.NewDecoder(response.Body).Decode(&body)
	if err != nil {
		logger.HavenLogger.Error().Err(err).Msg("Could not decode IP-API response.")
		return PublicLocation{}, err
	}

	return body, nil
}
