package main

import (
	"sync"

	"github.com/OverlayFox/VRC-Stream-Haven/libraries"
	"github.com/OverlayFox/VRC-Stream-Haven/logging"
	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"github.com/oschwald/geoip2-golang"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type serverHandler struct {
	server    *gortsplib.Server
	mutex     sync.Mutex
	stream    *gortsplib.ServerStream
	publisher *gortsplib.ServerSession
}

type Config struct {
	nodes []struct {
		publicIpAddress string
		publicPort      string
		latitude        float64
		longitude       float64
	}
}

var ipDb geoip2.Reader
var config Config
var logger = logging.Logger

func (sh *serverHandler) OnSessionClose(ctx *gortsplib.ServerHandlerOnSessionCloseCtx) {
	logger.Info("Session Closed")

	sh.mutex.Lock()
	defer sh.mutex.Unlock()

	if sh.stream != nil && ctx.Session == sh.publisher {
		sh.stream.Close()
		sh.stream = nil
	}
}

func (sh *serverHandler) OnSetup(ctx *gortsplib.ServerHandlerOnSetupCtx) (*base.Response, *gortsplib.ServerStream, error) {
	logger.Info("Setup Request received")

	if sh.stream == nil {
		return &base.Response{
			StatusCode: base.StatusNotFound,
		}, nil, nil
	}

	cSeq := ctx.Request.Header["CSeq"]

	node, err := libraries.LocateClient(ctx, config)

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
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		logger.WithFields(logrus.Fields{
			"error": err,
		}).Fatal("Could not read config file")
		return
	}

	if err := viper.Unmarshal(&config); err != nil {
		logger.WithFields(logrus.Fields{
			"error": err,
		}).Fatal("Error unmarshaling config")
		return
	}
}

func main() {
	handler := &serverHandler{}
	handler.server = &gortsplib.Server{
		Handler:           handler,
		RTSPAddress:       ":8554",
		UDPRTPAddress:     ":8000",
		UDPRTCPAddress:    ":8001",
		MulticastIPRange:  "224.1.0.0/16",
		MulticastRTPPort:  8002,
		MulticastRTCPPort: 8003,
	}

	logger.Info("Server is ready")
	panic(handler.server.StartAndWait())
}
