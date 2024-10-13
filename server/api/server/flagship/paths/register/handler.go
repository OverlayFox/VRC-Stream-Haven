package register

import (
	"encoding/json"
	"github.com/OverlayFox/VRC-Stream-Haven/api"
	"io"
	"net/http"

	"github.com/OverlayFox/VRC-Stream-Haven/harbor"
	"github.com/OverlayFox/VRC-Stream-Haven/logger"
	"github.com/OverlayFox/VRC-Stream-Haven/types"
)

// PostRegisterEscortToHaven adds the caller as an escort to the haven
func PostRegisterEscortToHaven(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	bodyJson, err := api.Decrypt(string(bodyBytes))
	if err != nil {
		http.Error(w, "Failed to decrypt body", http.StatusInternalServerError)
		return
	}

	var escort types.Escort
	if err := json.Unmarshal([]byte(bodyJson), &escort); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = harbor.Haven.RemoveEscort(escort.IpAddress)
	if err == nil {
		logger.HavenLogger.Warn().Msgf("Escort %s already exists, removing it", escort.IpAddress)
	}

	harbor.Haven.AddEscort(&escort)

	response := Response{
		IpAddress: harbor.Haven.Flagship.IpAddress.String(),
		SrtPort:   harbor.Haven.Flagship.SrtIngestPort,
	}

	responseJson, err := response.ToJson()
	if err != nil {
		http.Error(w, "Failed to parse response to json string", http.StatusInternalServerError)
		return
	}

	encrypt, err := api.Encrypt(responseJson)
	if err != nil {
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
