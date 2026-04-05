package main

import (
	"flag"
	"os"
	"os/signal"

	"github.com/OverlayFox/VRC-Stream-Haven/src/haven"
	"github.com/OverlayFox/VRC-Stream-Haven/src/logger"
	"github.com/OverlayFox/VRC-Stream-Haven/src/protocols/rtsp"
	"github.com/OverlayFox/VRC-Stream-Haven/src/protocols/srt"
	"github.com/rs/zerolog"
)

var (
	factory *logger.LoggerFactory
	log     zerolog.Logger
)

func main() {
	modePtr := flag.String("mode", "escort", "Mode to run the server in (flagship, escort)")
	flag.Parse()

	factory = logger.NewLoggerFactory(zerolog.DebugLevel, "logs")
	log = factory.NewLogger("main")

	if *modePtr == "flagship" {
		startFlagship()
	} else {
		startEscort()
	}
}

var tempPassphrase = "thisisaverysecurepassphrase"

func startFlagship() {
	log.Info().Msg("Starting in Flagship mode")

	haven, err := haven.NewHaven(tempPassphrase, "test", factory.NewLogger("haven"))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create Haven")
		return
	}

	rtspConfig := rtsp.Config{
		Port:       8554,
		Address:    "0.0.0.0",
		Passphrase: tempPassphrase,
		IsFlagship: false,
	}

	// Setup RTSP Server
	rtspServer, err := rtsp.New(rtspConfig, haven, factory.NewLogger("rtsp_server"))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create RTSP server")
		return
	}
	rtspServer.Start()

	srtConfig := srt.Config{
		Address: "0.0.0.0",
		Port:    8890,
	}
	srtServer, err := srt.New(factory.NewLogger("srt_server"), srtConfig, haven)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create SRT server")
		return
	}
	srtServer.Start()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig

	log.Info().Msg("Received interrupt signal, shutting down...")
}

func startEscort() {
	// haven := haven.NewHaven(tempPassphrase, "test", nil, factory.NewLogger("haven"))

	// // rtspConfig := rtsp.RtspConfig{
	// // 	Port:       8554,
	// // 	Address:    nil,
	// // 	Passphrase: tempPassphrase,
	// // 	IsFlagship: false,
	// // }

	// // rtspServer, err := rtsp.New(rtspConfig, haven, factory.NewLogger("rtsp_server"))
	// // if err != nil {
	// // 	log.Fatal().Err(err).Msg("Failed to create RTSP server")
	// // 	return
	// // }
	// // rtspServer.Start()

	// srtConfig := srt.SrtConfig{
	// 	Port:       8890,
	// 	Passphrase: tempPassphrase,
	// 	IsFlagship: false,
	// }
	// srtServer, err := srt.New(srtConfig, haven, factory.NewLogger("srt_server"))
	// if err != nil {
	// 	log.Fatal().Err(err).Msg("Failed to create SRT server")
	// 	return
	// }
	// err = srtServer.Call("localhost:8890")
	// if err != nil {
	// 	log.Fatal().Err(err).Msg("Failed to connect to SRT server")
	// }

	// sig := make(chan os.Signal, 1)
	// signal.Notify(sig, os.Interrupt)
	// <-sig

	// log.Info().Msg("Received interrupt signal, shutting down...")
}
