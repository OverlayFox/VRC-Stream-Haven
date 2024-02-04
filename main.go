package main

import (
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"github.com/oschwald/geoip2-golang"
	"github.com/kellydunn/golang-geo"
	"github.com/sirupsen/logrus"
)

var logger logrus.Logger
var ipDb geoip2.Reader

type Config struct {
	nodes []struct {
		publicIpAddress string
		publicPort      string
		CountryIso      string
		PostCode        string
	}
}

type serverHandler struct {
	s         *gortsplib.Server
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

func locateClient(ctx *gortsplib.ServerHandlerOnSetupCtx)(string) {
	var rerouteAddress string

	location, err := ipDb.City(net.ParseIP(ctx.Conn.NetConn().RemoteAddr().String()))
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err,
		}).Error("Could not locate IP-Address of incoming connection defaulting to base server")
	}
	fmt.Printf("Connection is coming from %s in %s\n", location.Postal.Code, location.Country.IsoCode)

	clientGeo := geo.NewPoint(location.Location.Latitude, location.Location.Longitude)
	nodeGeo := geoco
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
	h.s = &gortsplib.Server{
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
