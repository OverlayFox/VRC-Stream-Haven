package main

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/OverlayFox/VRC-Stream-Haven/src/geo"
	"github.com/OverlayFox/VRC-Stream-Haven/src/haven"
	"github.com/OverlayFox/VRC-Stream-Haven/src/protocols/rtsp"
	"github.com/OverlayFox/VRC-Stream-Haven/src/protocols/srt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "15:04:05"}).With().Str("component", "flagship").Timestamp().Logger()

func main() {
	logger.Info().Msg("Starting in Flagship mode")

	ctx, cancel := context.WithCancel(context.Background())

	geoConf := geo.Config{
		LicenseKey: "",
		AccountID:  "",
		Dir:        "./GeoDatabase",
	}
	locator, err := geo.NewLocator(ctx, logger, geoConf)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create geo locator")
		return
	}

	haven, err := haven.NewHaven(ctx, logger, locator, "thisisaverysecurepassphrase", "test")
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create haven")
		return
	}

	rtspConf := rtsp.Config{
		Port:           8554,
		Address:        "0.0.0.0",
		Passphrase:     "thisisaverysecurepassphrase",
		IsFlagship:     true,
		WriteTimeout:   10 * time.Second,
		WriteQueueSize: 8192,
	}
	rtspServer, err := rtsp.New(ctx, logger, rtspConf, haven, locator)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create RTSP server")
		return
	}
	rtspServer.Start()

	srtConfig := srt.Config{
		Address: "0.0.0.0",
		Port:    8890,
	}
	srtServer, err := srt.New(ctx, logger, srtConfig, haven, locator)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create SRT server")
		return
	}
	srtServer.Start()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig

	log.Info().Msg("Received interrupt signal, shutting down...")
	cancel()
}
