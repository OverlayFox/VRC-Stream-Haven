package main

import (
	"fmt"
	"github.com/OverlayFox/VRC-Stream-Haven/apiServer"
	"github.com/OverlayFox/VRC-Stream-Haven/apiService/escort"
	"github.com/OverlayFox/VRC-Stream-Haven/geoLocator"
	"github.com/OverlayFox/VRC-Stream-Haven/harbor"
	"github.com/OverlayFox/VRC-Stream-Haven/logger"
	"github.com/OverlayFox/VRC-Stream-Haven/rtspServer/flagship"
	"github.com/OverlayFox/VRC-Stream-Haven/streaming/ingest"
	"github.com/gorilla/mux"
	"github.com/oschwald/geoip2-golang"
	"net"
	"net/http"
	"os"
	"strconv"
)

type Config struct {
	IsFlagship bool
	Passphrase []byte
	ApiPort    int
	RtspPort   int
	SrtPort    int
	FlagshipIp net.IP
}

var config Config

func init() {
	logger.InitLogger()

	config = Config{
		Passphrase: getEnvPassphrase("PASSPHRASE", 10),
		ApiPort:    getEnvInt("API_PORT", 8080, 1, 65535),
		RtspPort:   getEnvInt("RTSP_PORT", 554, 1, 65535),
		SrtPort:    getEnvInt("SRT_PORT", 8554, 1, 65535),
		FlagshipIp: getEnvIP("FLAGSHIP_IP"),
	}

	if config.FlagshipIp == nil {
		config.IsFlagship = true
	}

	var err error
	geoLocator.GeoDatabase, err = geoip2.Open("./GeoLite2-City.mmdb")
	if err != nil {
		logger.HavenLogger.Fatal().Err(err).Msg("Failed to open GeoLite2-City.mmdb")
	}

	apiServer.Key = config.Passphrase
}

func getEnvPassphrase(key string, minLength int) []byte {
	value := os.Getenv(key)
	if len(value) < minLength {
		logger.HavenLogger.Fatal().Msgf("%s not set or shorter than %d characters.", key, minLength)
	}
	return []byte(value)
}

func getEnvInt(key string, defaultValue, min, max int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil || value < min || value > max {
		logger.HavenLogger.Warn().Msgf("%s was set to an invalid value, defaulting to %d", key, defaultValue)
		return defaultValue
	}
	return value
}

func getEnvIP(key string) net.IP {
	value := os.Getenv(key)
	if value == "" {
		return nil
	}
	return net.ParseIP(value)
}

func main() {
	var errChan chan error
	var err error

	// Start the API server
	var router *mux.Router
	if config.IsFlagship {
		router = apiServer.InitFlagshipApi()
		logger.HavenLogger.Info().Msgf("Starting API server as Flagship on %d", config.ApiPort)
	} else {
		router = apiServer.InitEscortApi()
		logger.HavenLogger.Info().Msgf("Starting API server as Escort on %d", config.ApiPort)
	}

	go func() {
		err := http.ListenAndServe(fmt.Sprintf(":%d", config.ApiPort), router)
		if err != nil {
			errChan <- err
		}
	}()

	// Generate a escort node
	node, err := harbor.MakeEscort(554)
	if err != nil {
		logger.HavenLogger.Fatal().Err(err).Msg("Failed to build Escort from local machine")
	}

	// Register the local machine with the flagship if the local machine is not the flagship
	if !config.IsFlagship {
		err = escort.RegisterEscortWithHaven(node, config.FlagshipIp)
		if err != nil {
			logger.HavenLogger.Fatal().Err(err).Msgf("Failed to register local machine with Flagship at IP: %s", config.FlagshipIp.String())
		}
	} else {
		harbor.MakeHaven(*node, uint16(config.SrtPort), string(config.Passphrase))
	}

	// Start the RTSP server
	flagship.ServerHandler = flagship.InitRtspServer(config.RtspPort)
	go func() {
		err = flagship.ServerHandler.Server.Start()
		if err != nil {
			errChan <- err
		}
	}()

	// Start the SRT ingest server
	if config.IsFlagship {
		err = ingest.InitFlagshipIngest()
	} else {
		err = ingest.InitEscortIngest()
	}

	select {
	case err := <-errChan:
		logger.HavenLogger.Fatal().Err(err).Msg("A fatal server error occurred")
	}
}
