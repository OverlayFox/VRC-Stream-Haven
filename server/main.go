package main

import (
	"github.com/OverlayFox/VRC-Stream-Haven/api"
	"github.com/OverlayFox/VRC-Stream-Haven/api/server"
	"github.com/OverlayFox/VRC-Stream-Haven/api/service/escort"
	"github.com/OverlayFox/VRC-Stream-Haven/geoLocator"
	"github.com/OverlayFox/VRC-Stream-Haven/harbor"
	"github.com/OverlayFox/VRC-Stream-Haven/logger"
	"github.com/OverlayFox/VRC-Stream-Haven/streaming/ingest"
	"github.com/OverlayFox/VRC-Stream-Haven/streaming/rtsp"
	"github.com/oschwald/geoip2-golang"
	"net"
	"net/http"
	"os"
	"strconv"
)

var isFlagship = false
var passphrase []byte
var apiPort = 8080
var rtspPort = 554
var srtPort = 8554
var flagshipIp net.IP

func init() {
	logger.InitLogger()

	if len(os.Getenv("PASSPHRASE")) < 10 {
		logger.HavenLogger.Fatal().Msg("PASSPHRASE not set or shorter than 10 characters.")
	}
	passphrase = []byte(os.Getenv("PASSPHRASE"))

	var err error
	rtspPort, err = strconv.Atoi(os.Getenv("RTSP_PORT"))
	if err != nil || rtspPort <= 0 || rtspPort > 65535 {
		logger.HavenLogger.Warn().Msg("RTSP_PORT was set to an invalid value, defaulting to 554")
		rtspPort = 554
	}

	apiPort, err = strconv.Atoi(os.Getenv("API_PORT"))
	if err != nil || apiPort <= 0 || apiPort > 65535 || apiPort == rtspPort {
		logger.HavenLogger.Warn().Msg("API_PORT was set to an invalid value, defaulting to 8080")
		apiPort = 8080
	}

	if os.Getenv("SRT_PORT") != "" {
		srtPort, err = strconv.Atoi(os.Getenv("SRT_PORT"))
		if err != nil || srtPort <= 0 || srtPort > 65535 || srtPort == rtspPort || srtPort == apiPort {
			logger.HavenLogger.Warn().Msg("SRT_PORT was set to an invalid value, defaulting to 8554")
			srtPort = 8554
		}
	}

	if os.Getenv("FLAGSHIP_IP") != "" {
		isFlagship = true
		flagshipIp = net.ParseIP(os.Getenv("FLAGSHIP_IP"))
	}

	geoLocator.GeoDatabase, err = geoip2.Open("./geoDatabase/GeoLite2-City.mmdb")
	if err != nil {
		logger.HavenLogger.Fatal().Err(err).Msg("Failed to open GeoLite2-City.mmdb")
	}

	api.Key = passphrase

}

func startFlagship() (chan error, error) {
	logger.HavenLogger.Info().Msg("Starting as Flagship")
	errChan := make(chan error)

	err := harbor.InitHaven()
	if err != nil {
		logger.HavenLogger.Fatal().Err(err).Msg("Failed to initialize Haven")
	}

	go func() {
		router := server.InitFlagshipApi()

		logger.HavenLogger.Info().Msg("Starting Flagship-API server on :8080")
		err := http.ListenAndServe(":8080", router)
		if err != nil {
			errChan <- err
		}
	}()

	rtsp.ServerHandler = rtsp.InitRtspServer(rtspPort)
	go func() {
		err = rtsp.ServerHandler.Server.Start()
		if err != nil {
			errChan <- err
		}
	}()

	return errChan, nil
}

func startEscort() (chan error, error) {
	logger.HavenLogger.Info().Msg("Starting as Escort")

	errChan := make(chan error)

	go func() {
		router := server.InitEscortApi()

		logger.HavenLogger.Info().Msg("Starting Escort-API server on :8080")
		err := http.ListenAndServe(":8080", router)
		if err != nil {
			errChan <- err
		}
	}()

	node, err := harbor.MakeEscort(554)
	if err != nil {
		return nil, err
	}

	err = escort.RegisterEscortWithHaven(node, flagshipIp)
	if err != nil {
		return nil, err
	}

	rtspServer := rtsp.InitRtspServer(rtspPort)
	go func() {
		err = rtspServer.Server.Start()
		if err != nil {
			errChan <- err
		}
	}()

	return errChan, nil
}

func main() {
	var errChan chan error
	var err error

	if isFlagship {
		errChan, err = startFlagship()
		if err != nil {
			logger.HavenLogger.Fatal().Err(err).Msg("A fatal server error occurred")
		}
	} else {
		errChan, err = startEscort()
		if err != nil {
			logger.HavenLogger.Fatal().Err(err).Msg("A fatal server error occurred")
		}
	}

	err = ingest.InitIngest(isFlagship)
	if err != nil {
		logger.HavenLogger.Fatal().Err(err).Msg("A fatal server error occurred. Could not initialize MediaMTX Ingest")
	}

	select {
	case err := <-errChan:
		logger.HavenLogger.Fatal().Err(err).Msg("A fatal server error occurred")
	}
}
