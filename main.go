package main

import (
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/base"
	geo "github.com/kellydunn/golang-geo"
	"github.com/oschwald/geoip2-golang"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var logger logrus.Logger
var ipDb geoip2.Reader

type Config struct {
	nodes []struct {
		publicIpAddress string
		publicPort      string
		latitude        float64
		longitude       float64
	}
}

var config Config

type serverHandler struct {
	server    *gortsplib.Server
	mutex     sync.Mutex
	stream    *gortsplib.ServerStream
	publisher *gortsplib.ServerSession
}

func (sh *serverHandler) OnSessionClose(ctx *gortsplib.ServerHandlerOnSessionCloseCtx) {
	log.Printf("session closed")

	sh.mutex.Lock()
	defer sh.mutex.Unlock()

	if sh.stream != nil && ctx.Session == sh.publisher {
		sh.stream.Close()
		sh.stream = nil
	}
}

func locateClient(ctx *gortsplib.ServerHandlerOnSetupCtx) *string {
	location, err := ipDb.City(net.ParseIP(ctx.Conn.NetConn().RemoteAddr().String()))
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err,
		}).Error("Could not locate IP-Address of incoming connection defaulting to base server")
		return nil
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

	return chosenNode.publicIpAddress + ":" + chosenNode.publicPort
}

func (sh *serverHandler) OnSetup(ctx *gortsplib.ServerHandlerOnSetupCtx) (*base.Response, *gortsplib.ServerStream, error) {
	log.Printf("setup request")

	if sh.stream == nil {
		return &base.Response{
			StatusCode: base.StatusNotFound,
		}, nil, nil
	}

	cSeq := ctx.Request.Header["CSeq"]

	if len(cSeq) <= 0 || rerouteAddress == "" || location.Postal.Code == "same as yours" {
		return &base.Response{
			StatusCode: base.StatusOK,
		}, sh.stream, nil
	}

	return &base.Response{
		StatusCode:    base.StatusFound,
		StatusMessage: "RTSP/2.0 302 Found closer node. Redirecting for load balancing",
		Header: base.Header{
			"CSeq":     []string{cSeq[0]},
			"Location": []string{rerouteAddress},
		},
	}, sh.stream, nil
}

func init() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	})

	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Error reading config file: %s\n", err)
		return
	}

	if err := viper.Unmarshal(&config); err != nil {
		fmt.Printf("Error unmarshaling config: %s\n", err)
		return
	}

	db, err := geoip2.Open("path/to/GeoLite2-Country.mmdb")
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err,
		}).Fatal("Could not load GeoLite Database")
		return
	}
	defer db.Close()
}

func main() {
	h := &serverHandler{}
	h.server = &gortsplib.Server{
		Handler:           h,
		RTSPAddress:       ":8554",
		UDPRTPAddress:     ":8000",
		UDPRTCPAddress:    ":8001",
		MulticastIPRange:  "224.1.0.0/16",
		MulticastRTPPort:  8002,
		MulticastRTCPPort: 8003,
	}

	log.Printf("server is ready")
	panic(h.s.StartAndWait())
}

func handleRTSPConnection(conn net.Conn) {
	defer conn.Close()

	ipAddress, err := ipDb.Country(net.ParseIP(conn.RemoteAddr().String()))
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err,
		}).Error("Could not locate IP-Address of incoming connection")
		return
	}
	fmt.Printf("Connection is coming from %s\n", ipAddress.Country.IsoCode)

	session := gortsplib.ServerSession()
	if err != nil {
		log.Println("Error creating RTSP session:", err)
		return
	}
}
