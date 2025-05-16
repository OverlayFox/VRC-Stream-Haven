package main

import (
	"flag"
	"os"
	"os/signal"

	"github.com/OverlayFox/VRC-Stream-Haven/src/api"
	"github.com/OverlayFox/VRC-Stream-Haven/src/governor"
	"github.com/OverlayFox/VRC-Stream-Haven/src/haven"
	"github.com/OverlayFox/VRC-Stream-Haven/src/logger"
	"github.com/OverlayFox/VRC-Stream-Haven/src/mediaServers/rtsp"
	"github.com/OverlayFox/VRC-Stream-Haven/src/mediaServers/srt"
	gosrt "github.com/datarhei/gosrt"
	"github.com/rs/zerolog"
)

var factory *logger.LoggerFactory
var log zerolog.Logger

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

func startFlagship() {
	tempPassphrase := "thisisaverysecurepassphrase"

	// Setup API
	apiRouter := api.NewRouter(factory.NewLogger("api"))
	go apiRouter.Start()

	packetBuffer := srt.NewPacketBuffer(1000, factory.NewLogger("packet_buffer"))

	// Setup Governor
	governor := governor.NewGovernor(factory.NewLogger("governor"))
	governor.AddHaven(haven.NewHaven(tempPassphrase, "haven", packetBuffer, factory.NewLogger("haven")))

	// Setup SRT Server
	srtConfig := gosrt.DefaultConfig()
	srtListener, err := gosrt.Listen("srt", "0.0.0.0:7710", srtConfig)
	if err != nil {
		log.Error().Err(err).Msg("Failed to start SRT listener")
		return
	}
	srtServer, err := srt.New(srtListener, governor, factory.NewLogger("srt_server"))
	if err != nil {
		log.Error().Err(err).Msg("Failed to create SRT server")
		return
	}
	srtServer.Start()

	// Setup RTSP Server
	rtspServer, err := rtsp.New(governor, 8554, true, tempPassphrase, factory.NewLogger("rtsp_server"))
	if err != nil {
		log.Error().Err(err).Msg("Failed to create RTSP server")
		return
	}
	rtspServer.Start()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig

	log.Info().Msg("Received interrupt signal, shutting down...")
}

func startEscort() {

}
