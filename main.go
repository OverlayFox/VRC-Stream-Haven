package main

import (
	"bufio"
	"fmt"
	"github.com/OverlayFox/VRC-Stream-Haven/haven"
	havenTypes "github.com/OverlayFox/VRC-Stream-Haven/haven/types"
	"github.com/OverlayFox/VRC-Stream-Haven/logger"
	"github.com/OverlayFox/VRC-Stream-Haven/servers/rtmp"
	"github.com/OverlayFox/VRC-Stream-Haven/servers/srt"
	"github.com/OverlayFox/VRC-Stream-Haven/servers/upnp"
	"os"
	"strings"
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

	go func() {
		srt.StartUpIngestSRT()
	}()

	go func() {
		rtsp
	}()

}

func asEscort() {
	logger.Log.Info().Msg("Starting as Escort...")

	upnp.SetupPortForward(0, 8557, 0)

	haven.MakeEscort()
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

	isFlagship := getShipState()

	if isFlagship {
		asFlagship()
	} else {
		asEscort()
	}

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
