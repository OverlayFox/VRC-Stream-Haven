package geoLocator

import (
	"log"
	"net"
	"strings"

	geo "github.com/kellydunn/golang-geo"
	"github.com/oschwald/geoip2-golang"
)

var GeoDatabase *geoip2.Reader

func LoadDatabase() *geoip2.Reader {
	database, err := geoip2.Open("./geoDatabase/GeoLite2-City.mmdb")
	if err != nil {
		log.Fatal(err)
	}

	return database
}

// LocateIp takes in an ipAddress, with or without port number, and converts it to the Latitude and Longitude.
// It may return an 0,0 if no country was found for the specified IP Address.
func LocateIp(ipAddress string) (float64, float64) {
	ip, _, err := net.SplitHostPort(ipAddress)
	if err != nil {
		if strings.Contains(err.Error(), "missing port in address") {
			ip = ipAddress
		} else {
			log.Fatal(err)
			return 0, 0
		}
	}

	record, err := GeoDatabase.City(net.ParseIP(ip))
	if err != nil {
		log.Fatal(err)
		return 0, 0
	}

	return record.Location.Latitude, record.Location.Longitude
}

// GetDistance takes in a latitude and longitude, and a list of nodes, and returns the closest node to the specified latitude and longitude.
func GetDistance(latitude float64, longitude float64, nodes []NodeStruct) NodeStruct {
	if latitude == 0 && longitude == 0 {
		// returns the first node, which is the Server itself
		return nodes[0]
	}

	clientLocation := geo.NewPoint(latitude, longitude)
	var closestNode NodeStruct
	var shortestDistance float64
	for i := 0; i < len(nodes); i++ {
		record, err := GeoDatabase.City(nodes[i].IpAddress)
		if err != nil {
			log.Fatal(err)
		}
		nodeLocation := geo.NewPoint(record.Location.Latitude, record.Location.Longitude)
		distance := clientLocation.GreatCircleDistance(nodeLocation)

		if closestNode.RtspEgressPort == 0 || closestNode.IpAddress == nil {
			closestNode = nodes[i]
			shortestDistance = distance
			continue
		}

		if shortestDistance > distance {
			closestNode = nodes[i]
			shortestDistance = distance
		}
	}

	return closestNode
}
