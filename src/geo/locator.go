package geo

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/OverlayFox/VRC-Stream-Haven/src/types"
)

type PublicLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

func (pl *PublicLocation) UnmarshalJSON(data []byte) error {
	type Format1 struct {
		Lat float64 `json:"lat"`
		Lon float64 `json:"lon"`
	}
	var f1 Format1
	err1 := json.Unmarshal(data, &f1)
	if err1 == nil && (f1.Lat != 0 || f1.Lon != 0) {
		pl.Latitude = f1.Lat
		pl.Longitude = f1.Lon
		return nil
	}

	type Format2 struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}
	var f2 Format2
	err2 := json.Unmarshal(data, &f2)
	if err2 == nil && (f2.Latitude != 0 || f2.Longitude != 0) {
		pl.Latitude = f2.Latitude
		pl.Longitude = f2.Longitude
		return nil
	}

	type Format3 struct {
		LatLong string `json:"loc"`
	}
	var f3 Format3
	err3 := json.Unmarshal(data, &f3)
	if err3 != nil {
		return fmt.Errorf("failed to parse location data: %v, %v", err1, err2)
	}
	coords := strings.Split(f3.LatLong, ",")
	if len(coords) != 2 {
		return fmt.Errorf("invalid location format: %s", f3.LatLong)
	}
	lat, err := strconv.ParseFloat(coords[0], 64)
	if err != nil {
		return fmt.Errorf("failed to parse latitude: %v", err)
	}
	long, err := strconv.ParseFloat(coords[1], 64)
	if err != nil {
		return fmt.Errorf("failed to parse longitude: %v", err)
	}
	pl.Latitude = lat
	pl.Longitude = long
	return nil
}

// GetPublicLocation Gets the Public IP Address of the Server running this function.
func GetPublicLocation(addr net.Addr) (types.Location, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	ip, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		return types.Location{}, fmt.Errorf("failed to parse IP address: %v", err)
	}
	parsedIp := net.ParseIP(ip)

	var locatorUris []string
	if parsedIp.IsPrivate() || parsedIp.IsLoopback() || parsedIp.IsMulticast() || parsedIp.IsUnspecified() {
		locatorUris = []string{
			"http://ip-api.com/json/",
			"http://ipwho.is/",
			"https://ipinfo.io/json",
		}
	} else {
		locatorUris = []string{
			fmt.Sprintf("http://ip-api.com/json/%s", ip),
			fmt.Sprintf("http://ipwho.is/%s", ip),
			fmt.Sprintf("https://ipinfo.io/%s/json", ip),
		}
	}

	ipApiFunc := func(uri string) (types.Location, error) {
		response, err := client.Get(uri)
		if err != nil {
			return types.Location{}, err
		}
		defer response.Body.Close()

		bodyBytes, err := io.ReadAll(response.Body)
		if err != nil {
			return types.Location{}, fmt.Errorf("failed to read response body: %v", err)
		}

		var body PublicLocation
		err = json.Unmarshal(bodyBytes, &body)
		if err != nil {
			return types.Location{}, fmt.Errorf("failed to decode response body: %v, %s", err, string(bodyBytes))
		}

		if body.Latitude == 0 && body.Longitude == 0 {
			return types.Location{}, fmt.Errorf("received zero coordinates from '%s'", uri)
		}

		return types.Location{
			Latitude:  body.Latitude,
			Longitude: body.Longitude,
		}, nil
	}

	for _, uri := range locatorUris {
		location, err := ipApiFunc(uri)
		if err == nil {
			return location, nil
		}
	}

	return types.Location{}, fmt.Errorf("failed to get public location: primary and fallback services failed: '%v'", err)
}
