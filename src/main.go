package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/OverlayFox/VRC-Stream-Haven/depreciated"
	"github.com/OverlayFox/VRC-Stream-Haven/escort/apiServer"
	rtspEscort "github.com/OverlayFox/VRC-Stream-Haven/escort/rtspServer"
	rtspFlagship "github.com/OverlayFox/VRC-Stream-Haven/flagship/rtspServer"
	apiServer2 "github.com/OverlayFox/VRC-Stream-Haven/shared/apiServer"
	"github.com/OverlayFox/VRC-Stream-Haven/shared/crypto"
	"github.com/OverlayFox/VRC-Stream-Haven/shared/geoLocator"
	"github.com/OverlayFox/VRC-Stream-Haven/shared/logger"
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
	IsFlagship      bool
	Passphrase      []byte
	ApiPort         int
	RtspPort        int
	SrtPort         int
	FlagshipIp      net.IP
	BackendIp       net.IP
	FlagshipApiPort int
}

var config Config

func init() {
	logger.InitLogger()

	config = Config{
		Passphrase:      getEnvPassphrase("PASSPHRASE", 10),
		ApiPort:         getEnvInt("API_PORT", 8080, 1, 65535),
		RtspPort:        getEnvInt("RTSP_PORT", 554, 1, 65535),
		SrtPort:         getEnvInt("SRT_PORT", 8554, 1, 65535),
		FlagshipIp:      getEnvIP("FLAGSHIP_IP"),
		FlagshipApiPort: getEnvInt("FLAGSHIP_API_PORT", 8080, 1, 65535),
		BackendIp:       getEnvIP("BACKEND_IP"),
	}

	if config.FlagshipIp == nil {
		config.IsFlagship = true
	}

	var err error
	geoLocator.GeoDatabase, err = geoip2.Open("./GeoLite2-City.mmdb")
	if err != nil {
		logger.HavenLogger.Fatal().Err(err).Msg("Failed to open GeoLite2-City.mmdb")
	}

	crypto.Key = config.Passphrase

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

	// Start the API src
	var router *mux.Router
	if config.IsFlagship {
		router = apiServer2.InitFlagshipApi()
		logger.HavenLogger.Info().Msgf("Started API src as Flagship on %d", config.ApiPort)
	} else {
		router = apiServer2.InitEscortApi()
		logger.HavenLogger.Info().Msgf("Started API src as Escort on %d", config.ApiPort)
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
	node, err := depreciated.MakeEscort(uint16(config.RtspPort), uint16(config.ApiPort), config.BackendIp)
	if err != nil {
		logger.HavenLogger.Fatal().Err(err).Msg("Failed to build Escort from local machine")
	}

	// Register the local machine with the flagship if the local machine is not the flagship
	if !config.IsFlagship {
		err = apiServer.RegisterEscortWithHaven(node, config.FlagshipIp, config.FlagshipApiPort, config.Passphrase)
		if err != nil {
			logger.HavenLogger.Fatal().Err(err).Msgf("Failed to register local machine with Flagship at IP: %s", config.FlagshipIp.String())
		}
		logger.HavenLogger.Info().Msgf("Registered local machine with Flagship at IP: %s", config.FlagshipIp.String())
	} else {
		depreciated.MakeHaven(*node, uint16(config.SrtPort), string(config.Passphrase))

		logger.HavenLogger.Info().Msgf("Initilised Haven. Local machine is now the Flagship")
	}

	// Start the RTSP src
	if !config.IsFlagship {
		rtspEscort.ServerHandler = rtspEscort.InitRtspServer(config.RtspPort, string(config.Passphrase))

		go func() {
			errChan <- rtspEscort.ServerHandler.Server.StartAndWait()
		}()
		logger.HavenLogger.Info().Msgf("Started RTSP src as Escort on %d", config.RtspPort)
	} else {
		rtspFlagship.ServerHandler = rtspFlagship.InitRtspServer(config.RtspPort, string(config.Passphrase))

		go func() {
			errChan <- rtspFlagship.ServerHandler.Server.StartAndWait()
		}()
		logger.HavenLogger.Info().Msgf("Started RTSP src as Flagship on %d", config.RtspPort)
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	// Start the SRT ingest src
	if config.IsFlagship {
		err = depreciated.InitFlagshipIngest(config.SrtPort, config.RtspPort)

		logger.HavenLogger.Info().Msgf("Started SRT src on %d as Flagship. Ready to receive ingest signal.", config.SrtPort)
	} else {
		err = depreciated.InitEscortIngest(config.SrtPort, config.RtspPort)

		logger.HavenLogger.Info().Msgf("Started SRT src on port %d. Pulling SRT Feed from Flagship", config.SrtPort)
	}

	select {
	case err := <-errChan:
		logger.HavenLogger.Fatal().Err(err).Msg("A fatal src error occurred")
	case <-signalChan:
		logger.HavenLogger.Info().Msg("Received termination signal, shutting down")

		ctx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		if err := apiSrv.Shutdown(ctx); err != nil {
			logger.HavenLogger.Error().Err(err).Msg("Failed to shut down API src gracefully")
		} else {
			logger.HavenLogger.Info().Msg("API src shut down gracefully")
		}

		if rtspFlagship.ServerHandler != nil {
			rtspFlagship.ServerHandler.Server.Close()
			logger.HavenLogger.Info().Msg("Flagship RTSP src shut down gracefully")
		}

		if rtspEscort.ServerHandler != nil {
			rtspEscort.ServerHandler.Server.Close()
			logger.HavenLogger.Info().Msg("Escort RTSP src shut down gracefully")
		}

		err = depreciated.StopMediaMtx()
		if err != nil {
			logger.HavenLogger.Error().Err(err).Msg("Failed to stop mediaMTX")
		} else {
			logger.HavenLogger.Info().Msg("Stopped mediaMTX")
		}

	}
}
