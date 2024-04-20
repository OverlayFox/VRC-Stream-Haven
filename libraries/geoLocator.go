package libraries

import (
	"github.com/oschwald/geoip2-golang"
	"log"
	"net"
	"strings"
)

// LocateIp takes in an ipAddress, with or without port number, and converts it to a Country ISO Code.
// It may return an empty string if no country was found for the specified IP Address
func LocateIp(ipAddress string) string {
	database, err := geoip2.Open("./GeoDatabase/GeoLite2-Country.mmdb")
	if err != nil {
		log.Fatal(err)
		return ""
	}
	defer database.Close()

	ip, _, err := net.SplitHostPort(ipAddress)
	if err != nil {
		if strings.Contains(err.Error(), "missing port in address") {
			ip = ipAddress
		} else {
			log.Fatal(err)
			return ""
		}
	}

	record, err := database.Country(net.ParseIP(ip))
	if err != nil {
		log.Fatal(err)
		return ""
	}

	return record.Country.IsoCode
}
