package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"time"

	_ "net/http/pprof"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"

	"github.com/OverlayFox/VRC-Stream-Haven/src/haven"
	"github.com/OverlayFox/VRC-Stream-Haven/src/protocols/hls"
	"github.com/OverlayFox/VRC-Stream-Haven/src/protocols/srt"
)

var logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "15:04:05"}).With().Str("component", "escort").Timestamp().Logger()

func startPprof(port string) {
	go func() {
		logger.Info().Str("port", port).Msg("Starting pprof server")
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			logger.Error().Err(err).Msg("pprof server error")
		}
	}()
}

func loadEnv() {
	err := godotenv.Load()
	if err != nil {
		logger.Panic().Err(err).Msg("Error loading .env file")
	}
}

func main() {
	logger.Info().Msg("Starting in Escort mode")
	loadEnv()

	pprofPort := os.Getenv("PPROF_PORT_ESCORT")
	if pprofPort != "" {
		startPprof(pprofPort)
	}

	ctx, cancel := context.WithCancel(context.Background())
	haven, err := haven.NewHaven(ctx, logger, nil, "thisisaverysecurepassphrase", "test")
	if err != nil {
		logger.Panic().Err(err).Msg("Failed to create haven")
		return
	}

	hlsConf := hls.Config{
		Port:           8555,
		Address:        "0.0.0.0",
		Passphrase:     "thisisaverysecurepassphrase",
		IsFlagship:     false,
		WriteTimeout:   10 * time.Second,
		WriteQueueSize: 8192,
	}
	hlsServer, err := hls.New(ctx, logger, hlsConf, haven, nil)
	if err != nil {
		logger.Panic().Err(err).Msg("Failed to create HLS server")
		return
	}
	hlsServer.Start()

	srtConfig := srt.Config{
		Address: "0.0.0.0",
		Port:    8891,
	}
	srtServer, err := srt.New(ctx, logger, srtConfig, haven, nil)
	if err != nil {
		logger.Panic().Err(err).Msg("Failed to create SRT server")
		return
	}

	err = srtServer.Dial("127.0.0.1:8890", "escort:ingest", "thisisaverysecurepassphrase")
	if err != nil {
		logger.Panic().Err(err).Msg("Failed to dial SRT server")
		return
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig

	logger.Info().Msg("Received interrupt signal, shutting down...")
	cancel()
}
