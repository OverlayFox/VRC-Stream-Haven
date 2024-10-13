package info

import (
	"github.com/OverlayFox/VRC-Stream-Haven/api"
	"github.com/OverlayFox/VRC-Stream-Haven/logger"
	"github.com/OverlayFox/VRC-Stream-Haven/streaming/rtsp"
	"net/http"
	"os"
	"strconv"
)

// GetInfo returns the information of the escort to the caller.
func GetInfo(w http.ResponseWriter, r *http.Request) {
	currentViewers, err := rtsp.ServerHandler.GetReaders()
	if err != nil {
		logger.HavenLogger.Error().Err(err).Msg("Could not get amount of Readers from external Stream Struct")
		http.Error(w, "Could not get amount of Readers", http.StatusInternalServerError)
		return
	}

	var maxViewers int
	if os.Getenv("MAX_VIEWERS") != "" && os.Getenv("MAX_VIEWERS") != "0" {
		maxViewers, err = strconv.Atoi(os.Getenv("MAX_VIEWERS"))
		if err != nil {
			maxViewers = 0
		}
	}

	response := Response{
		CurrentViewers:    currentViewers,
		MaxAllowedViewers: maxViewers,
	}

	responseJson, err := response.ToJson()
	if err != nil {
		logger.HavenLogger.Error().Err(err).Msg("Could not parse response to json string")
		http.Error(w, "Could not parse response to json string", http.StatusInternalServerError)
		return
	}

	encrypt, err := api.Encrypt(responseJson)
	if err != nil {
		logger.HavenLogger.Error().Err(err).Msg("Failed to encrypt response")
		http.Error(w, "Failed to encrypt response", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte(encrypt))
	if err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
		return
	}
}
