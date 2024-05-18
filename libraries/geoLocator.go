package libraries

import (
	geo "github.com/kellydunn/golang-geo"
	"github.com/oschwald/geoip2-golang"
	"log"
	"net"
	"strings"
)

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
	database := LoadDatabase()
	defer database.Close()

	ip, _, err := net.SplitHostPort(ipAddress)
	if err != nil {
		if strings.Contains(err.Error(), "missing port in address") {
			ip = ipAddress
		} else {
			log.Fatal(err)
			return 0, 0
		}
	}

	record, err := database.City(net.ParseIP(ip))
	if err != nil {
		log.Fatal(err)
		return 0, 0
	}

	return record.Location.Latitude, record.Location.Longitude

}

func GetDistance(latitude float64, longitude float64, nodes []NodeStruct) NodeStruct {
	database := LoadDatabase()
	defer database.Close()
	clientLocation := geo.NewPoint(latitude, longitude)

	var closestNode NodeStruct
	var shortestDistance float64
	for i := 0; i < len(nodes); i++ {
		record, err := database.City(net.ParseIP(nodes[i].IpAddress))
		if err != nil {
			log.Fatal(err)
		}
		nodeLocation := geo.NewPoint(record.Location.Latitude, record.Location.Longitude)
		distance := clientLocation.GreatCircleDistance(nodeLocation)

		if (NodeStruct{}) == closestNode {
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
