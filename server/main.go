package main

import (
	"github.com/OverlayFox/VRC-Stream-Haven/api/escort"
	"github.com/OverlayFox/VRC-Stream-Haven/streaming/ingest"
	"github.com/OverlayFox/VRC-Stream-Haven/streaming/rtsp"
	"net/http"
	"os"
	"strings"

	"github.com/OverlayFox/VRC-Stream-Haven/api"
	"github.com/OverlayFox/VRC-Stream-Haven/harbor"
	"github.com/OverlayFox/VRC-Stream-Haven/logger"
)

func startFlagship() (chan error, error) {
	logger.HavenLogger.Info().Msg("Starting as Flagship")

	errChan := make(chan error)
	err := harbor.InitHaven(8890, 554)
	if err != nil {
		logger.HavenLogger.Fatal().Err(err).Msg("Failed to initialize Haven")
	}

	go func() {
		router := api.InitFlagshipApi()

		logger.HavenLogger.Info().Msg("Starting Flagship-API server on :8080")
		err := http.ListenAndServe(":8080", router)
		if err != nil {
			errChan <- err
		}
	}()

	go func() {
		err = rtsp.ServerHandler.Server.Start()
		if err != nil {
			errChan <- err
		}
	}()

	err = ingest.InitIngest(false)
	if err != nil {
		return nil, err
	}

	return errChan, nil
}

func startEscort() (chan error, error) {
	logger.HavenLogger.Info().Msg("Starting as Escort")

	errChan := make(chan error)

	go func() {
		router := api.InitEscortApi()

		logger.HavenLogger.Info().Msg("Starting Escort-API server on :8080")
		err := http.ListenAndServe(":8080", router)
		if err != nil {
			errChan <- err
		}
	}()

	node, err := harbor.MakeEscort(554)
	if err != nil {
		return nil, err
	}

	err = escort.RegisterEscortWithHaven(node)
	if err != nil {
		return nil, err
	}

	go func() {
		err = rtsp.ServerHandler.Server.Start()
		if err != nil {
			errChan <- err
		}
	}()

	err = ingest.InitIngest(true)
	if err != nil {
		return nil, err
	}

	return errChan, nil
}

func main() {
	var errChan chan error
	var err error

	if len(os.Getenv("PASSPHRASE")) < 10 {
		logger.HavenLogger.Fatal().Msg("PASSPHRASE must be at least 10 characters long")
	}

	if strings.ToUpper(os.Getenv("IS_NODE")) == "FALSE" {
		errChan, err = startFlagship()
		if err != nil {
			logger.HavenLogger.Fatal().Err(err).Msg("A fatal server error occurred")
		}
	} else {
		errChan, err = startEscort()
		if err != nil {
			logger.HavenLogger.Fatal().Err(err).Msg("A fatal server error occurred")
		}
	}

	select {
	case err := <-errChan:
		logger.HavenLogger.Fatal().Err(err).Msg("A fatal server error occurred")
	}
}
