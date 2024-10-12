package main

import (
	"net/http"
	"os"
	"strings"

	"github.com/OverlayFox/VRC-Stream-Haven/api"
	"github.com/OverlayFox/VRC-Stream-Haven/harbor"
	"github.com/OverlayFox/VRC-Stream-Haven/logger"
	"github.com/OverlayFox/VRC-Stream-Haven/types"
)

func startFlagship() (chan error, error) {
	logger.HavenLogger.Info().Msg("Starting as Flagship")

	errChan := make(chan error)

	escort, err := harbor.MakeEscort(554)
	if err != nil {
		return nil, err
	}
	flagship := harbor.MakeFlagship(escort, 2088, "ingest")
	harbor.MakeHaven(&[]*types.Escort{escort}, flagship, true)

	go func() {
		router := api.InitApi(false)

		logger.HavenLogger.Info().Msg("Starting API server on :8080")
		err := http.ListenAndServe(":8080", router)
		if err != nil {
			errChan <- err
		}
	}()

	return errChan, nil
}

func startEscort() (chan error, error) {
	logger.HavenLogger.Info().Msg("Starting as Escort")

	errChan := make(chan error)

	go func() {
		router := api.InitApi(true)

		logger.HavenLogger.Info().Msg("Starting API server on :8080")
		err := http.ListenAndServe(":8080", router)
		if err != nil {
			errChan <- err
		}
	}()

	escort, err := harbor.MakeEscort(554)
	if err != nil {
		return nil, err
	}

	err = api.RegisterEscort(escort)
	if err != nil {
		return nil, err
	}

	go func() {
		// @ToDo: Implement start of RTSP Server
	}()

	go func() {

	}()

	return errChan, nil
}

func main() {
	var errChan chan error
	var err error

	if strings.ToUpper(os.Getenv("IS_NODE")) == "FALSE" {
		errChan, err = startFlagship()
		if err != nil {
			return
		}
	} else {
		errChan, err = startEscort()
		if err != nil {
			return
		}
	}

	select {
	case err := <-errChan:
		logger.HavenLogger.Fatal().Err(err).Msg("A fatal server error occurred")
	}

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
