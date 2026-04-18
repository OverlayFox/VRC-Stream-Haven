package main

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"

	"github.com/OverlayFox/VRC-Stream-Haven/src/haven"
	"github.com/OverlayFox/VRC-Stream-Haven/src/protocols/rtsp"
	"github.com/OverlayFox/VRC-Stream-Haven/src/protocols/srt"
)

var logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "15:04:05"}).With().Str("component", "escort").Timestamp().Logger()

func loadEnv() {
	err := godotenv.Load()
	if err != nil {
		logger.Panic().Err(err).Msg("Error loading .env file")
	}
}

func main() {
	logger.Info().Msg("Starting in Escort mode")

	loadEnv()

	ctx, cancel := context.WithCancel(context.Background())
	haven, err := haven.NewHaven(ctx, logger, nil, "thisisaverysecurepassphrase", "test")
	if err != nil {
		logger.Panic().Err(err).Msg("Failed to create haven")
		return
	}

	rtspConf := rtsp.Config{
		Port:           8555,
		Address:        "0.0.0.0",
		Passphrase:     "thisisaverysecurepassphrase",
		IsFlagship:     false,
		WriteTimeout:   10 * time.Second,
		WriteQueueSize: 8192,
	}
	rtspServer, err := rtsp.New(ctx, logger, rtspConf, haven, nil)
	if err != nil {
		logger.Panic().Err(err).Msg("Failed to create RTSP server")
		return
	}
	rtspServer.Start()

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
