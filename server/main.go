package main

import (
	"context"
	"errors"
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
	"os/signal"
	"strconv"
	"syscall"
	"time"
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

	logger.HavenLogger.Info().Msg("User configuration loaded successfully")
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
	var errChan = make(chan error, 1)

	// Start the API server
	var router *mux.Router
	if config.IsFlagship {
		router = apiServer.InitFlagshipApi()
		logger.HavenLogger.Info().Msgf("Started API server as Flagship on %d", config.ApiPort)
	} else {
		router = apiServer.InitEscortApi()
		logger.HavenLogger.Info().Msgf("Started API server as Escort on %d", config.ApiPort)
	}

	apiSrv := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.ApiPort),
		Handler: router,
	}

	go func() {
		if err := apiSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
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

		logger.HavenLogger.Info().Msgf("Registered local machine with Flagship at IP: %s", config.FlagshipIp.String())
	} else {
		harbor.MakeHaven(*node, uint16(config.SrtPort), string(config.Passphrase))

		logger.HavenLogger.Info().Msgf("Initilised Haven. Local machine is now the Flagship")
	}

	// Start the RTSP server
	flagship.ServerHandler = flagship.InitRtspServer(config.RtspPort)
	go func() {
		errChan <- flagship.ServerHandler.Server.StartAndWait()
	}()
	logger.HavenLogger.Info().Msgf("Started RTSP server on %d", config.RtspPort)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	// Start the SRT ingest server
	if config.IsFlagship {
		err = ingest.InitFlagshipIngest(config.SrtPort)

		logger.HavenLogger.Info().Msgf("Started SRT server on %d as Flagship.", config.SrtPort)
	} else {
		err = ingest.InitEscortIngest(config.SrtPort)

		logger.HavenLogger.Info().Msgf("Started SRT server on %d as Escort.", config.SrtPort)
	}

	select {
	case err := <-errChan:
		logger.HavenLogger.Fatal().Err(err).Msg("A fatal server error occurred")
	case <-signalChan:
		logger.HavenLogger.Info().Msg("Received termination signal, shutting down")

		ctx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		if err := apiSrv.Shutdown(ctx); err != nil {
			logger.HavenLogger.Error().Err(err).Msg("Failed to shut down API server gracefully")
		} else {
			logger.HavenLogger.Info().Msg("API server shut down gracefully")
		}

		flagship.ServerHandler.Server.Close()
		logger.HavenLogger.Info().Msg("RTSP server shut down gracefully")

		err = ingest.StopMediaMtx()
		if err != nil {
			logger.HavenLogger.Error().Err(err).Msg("Failed to stop mediaMTX")
		} else {
			logger.HavenLogger.Info().Msg("Stopped mediaMTX")
		}

	}
}
