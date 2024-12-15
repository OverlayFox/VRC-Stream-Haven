package main

import (
	"context"
	"fmt"
	"github.com/OverlayFox/VRC-Stream-Haven/depreciated"
	"github.com/OverlayFox/VRC-Stream-Haven/escort/apiServer"
	escortApi "github.com/OverlayFox/VRC-Stream-Haven/escort/apiServer"
	rtspEscort "github.com/OverlayFox/VRC-Stream-Haven/escort/rtspServer"
	flagshipApi "github.com/OverlayFox/VRC-Stream-Haven/flagship/apiServer"
	rtspFlagship "github.com/OverlayFox/VRC-Stream-Haven/flagship/rtspServer"
	"github.com/OverlayFox/VRC-Stream-Haven/shared/config"
	"github.com/OverlayFox/VRC-Stream-Haven/shared/geoLocator"
	"github.com/OverlayFox/VRC-Stream-Haven/shared/logger"
	"github.com/OverlayFox/VRC-Stream-Haven/shared/overseer"
	"github.com/oschwald/geoip2-golang"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cfg := logger.Config{
		LogFilePath: "logs/app.log",
		LogLevel:    logger.InfoLevel,
		AppVersion:  os.Getenv("APP_VERSION"),
		Environment: "production",
		UseConsole:  true,
		UseJSON:     false,
	}

	if err := logger.Init(cfg); err != nil {
		panic(err)
	}
	defer logger.Shutdown()

	log := logger.Get()
	log.Info().Msg("Initialised Logger")

	conf, err := config.CreateConfigFromEnv()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create config from env")
	} else {
		log.Info().Msg("Created config from env")
	}

	geoLocator.GeoDatabase, err = geoip2.Open("./GeoLite2-City.mmdb")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load GeoLite2-City.mmdb")
	} else {
		log.Info().Msg("Loaded GeoLite2-City.mmdb")
	}

	currentLocation, err := geoLocator.GetCurrentPublicLocation()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get current public location")
	}

	escort := overseer.MakeEscort(uint16(conf.RtspPort), uint16(conf.ApiPort), currentLocation)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to build Escort from local machine")
	}

	var errChan = make(chan error, 1)
	apiLog := logger.Named("apiServer")
	if conf.IsFlagship {
		var server = flagshipApi.NewFlagshipApiServer(apiLog, conf.Passphrase, nil)

		go func() {
			if err := server.Start(fmt.Sprintf(":%d", conf.ApiPort)); err != nil {
				errChan <- err
			}
		}()

		log.Info().Msgf("Started API server as Flagship on port %d", conf.ApiPort)
	} else {
		var server = escortApi.NewEscortApiServer(apiLog, conf.Passphrase)

		go func() {
			if err := server.Start(fmt.Sprintf(":%d", conf.ApiPort)); err != nil {
				errChan <- err
			}
		}()

		log.Info().Msgf("Started API server as Escort on port %d", conf.ApiPort)
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
