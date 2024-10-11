package main

import (
	"bufio"
	"fmt"
	"github.com/OverlayFox/VRC-Stream-Haven/ingest/rtmp"
	"github.com/OverlayFox/VRC-Stream-Haven/ingest/srt"
	"github.com/OverlayFox/VRC-Stream-Haven/server/haven"
	havenTypes "github.com/OverlayFox/VRC-Stream-Haven/server/haven/types"
	"github.com/OverlayFox/VRC-Stream-Haven/server/logger"
	"github.com/OverlayFox/VRC-Stream-Haven/server/servers/upnp"
	"os"
	"strings"
	"time"
)

func asFlagship() {
	logger.Log.Info().Msg("Starting as Flagship...")

	var srtIngestPort uint16 = 9710
	var rtmpIngestPort uint16 = 1935
	var rtspEgressPort uint16 = 8557
	var apiPort uint16 = 8042

	upnp.SetupPortForward(srtIngestPort, rtspEgressPort, apiPort)

	escort, err := haven.MakeEscort(rtspEgressPort)
	if err != nil {
		logger.Log.Fatal().Err(err).Msg("Could not create Flagship as Escort")
	}

	flagship := haven.MakeFlagship(escort, apiPort, srtIngestPort, rtmpIngestPort)

	haven.MakeHaven(&[]*havenTypes.Escort{escort}, flagship, true)

	rtmp.StartRtmpServer()
}

func asEscort() {
	logger.Log.Info().Msg("Starting as Escort...")

	upnp.SetupPortForward(0, 8557, 0)

	//haven.MakeEscort()
}

func getShipState() bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Are you the Flagship? (y/n): ")
	input, err := reader.ReadString('\n')
	if err != nil {
		logger.Log.Fatal().Err(err).Msg("Could not read input")
	}

	userOutput := strings.ToUpper(strings.TrimSpace(input))

	if userOutput != "Y" && userOutput != "N" {
		fmt.Print("Invalid input. Please enter 'y' or 'n'\n")
		return getShipState()
	}

	if userOutput == "Y" {
		return true
	}

	return false
}

func main() {
	logger.InitLogger()

	server := &srt.Server{
		Address:           "127.0.0.1:6001",
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		WriteQueueSize:    512,
		UDPMaxPayloadSize: 1472,
		Logger:            &logger.Log,
	}

	err := server.Initialize()
	if err != nil {
		logger.Log.Fatal().Err(err).Msg("failed to initialize SRT server")
	}

	select {}

	//isFlagship := getShipState()
	//
	//if isFlagship {
	//	asFlagship()
	//} else {
	//	asEscort()
	//}

	//lib.Scanner = bufio.NewScanner(os.Stdin)
	//if lib.IsServer() {
	//	if lib.IsIngestTypeSrt() {
	//		ingestSrtServer := servers.SetupIngestSrt(lib.GetSrtIngestPort())
	//	} else {
	//		ingestRtmpServer := servers.SetupIngestRtmp(lib.GetRtmpIngestPort())
	//	}
	//
	//	Config.Server.RtspEgressPort = lib.GetRtspEgressPort(true)
	//	Config.Server.IpAddress = lib.GetPublicIpAddress()
	//	Config.Nodes = append(Config.Nodes, lib.GetNodes()...)
	//	Config.Server.Passphrase = lib.GenerateKey()
	//	backendSrtServer := servers.SetupBackendSrt(Config.Server.Passphrase)
	//
	//} else {
	//
	//}

	//servers.StartUpIngestSRT()

	//lib.GeoDatabase = lib.LoadDatabase()
	//lib.InitialiseConfig()

	//rtspHandler := &servers.RtspServerHandler{}
	//rtspHandler.Server = &gortsplib.Server{
	//	Handler:     rtspHandler,
	//	RTSPAddress: lib.Config.Server.RtspStreamingPortString(),
	//}
	//go rtspHandler.Server.StartAndWait()
	//
	//if os.Getenv("IS_NODE") == "False" {
	//	go servers.StartRtmpServer()
	//}
	//
	//select {}
}
