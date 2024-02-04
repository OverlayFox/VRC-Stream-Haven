package libraries

import (
	"fmt"
	"net"

	"github.com/bluenviron/gortsplib/v4"
	geo "github.com/kellydunn/golang-geo"
	"github.com/oschwald/geoip2-golang"
    "github.com/sirupsen/logrus"
    "github.com/OverlayFox/VRC-Stream-Haven/logging"    
)

type Config struct {
	nodes []struct {
		publicIpAddress string
		publicPort      string
		latitude        float64
		longitude       float64
	}
}

var logger = logging.Logger
var ipDb geoip2.Reader


func LocateClient(ctx *gortsplib.ServerHandlerOnSetupCtx, config Config) (string, error) {
    db, err := geoip2.Open("path/to/GeoLite2-Country.mmdb")
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err,
		}).Fatal("Could not load GeoLite Database")
		return "", err
	}
	defer db.Close()

	location, err := ipDb.City(net.ParseIP(ctx.Conn.NetConn().RemoteAddr().String()))
	if err != nil {
		return "", err
	}
	fmt.Printf("Connection is coming from %s in %s\n", location.Postal.Code, location.Country.IsoCode)

	clientGeo := geo.NewPoint(location.Location.Latitude, location.Location.Longitude)

	var distance float64
	var chosenNode struct {
		publicIpAddress string
		publicPort      string
		latitude        float64
		longitude       float64
	}
	for _, node := range config.nodes {
		calculatedDistance := clientGeo.GreatCircleDistance(geo.NewPoint(node.latitude, node.longitude))

		if distance <= 0 || distance > calculatedDistance {
			distance = calculatedDistance
			chosenNode = node
			continue
		}
	}

	return chosenNode.publicIpAddress + ":" + chosenNode.publicPort, nil
}
