package flagship

import (
	"encoding/json"
	"github.com/OverlayFox/VRC-Stream-Haven/api"
	"github.com/OverlayFox/VRC-Stream-Haven/harbor"
	"github.com/OverlayFox/VRC-Stream-Haven/logger"
	"github.com/OverlayFox/VRC-Stream-Haven/types"
	"io"
	"net"
	"net/http"
)

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
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

	var body RegisterBody
	if err := json.Unmarshal([]byte(bodyJson), &body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = harbor.Haven.RemoveEscort(body.Username)
	if err == nil {
		logger.HavenLogger.Warn().Msgf("Escort %s already exists, removing it", body.Username)
	}

	harbor.Haven.AddEscort(&types.Escort{
		IpAddress:      net.ParseIP(body.IpAddress),
		RtspEgressPort: body.RtspEgressPort,
		Latitude:       body.Latitude,
		Longitude:      body.Longitude,
		Username:       body.Username,
	})

	response := RegisterResponse{
		Success:     true,
		IpAddress:   harbor.Haven.Flagship.Ship.IpAddress.String(),
		Port:        harbor.Haven.Flagship.SrtIngestPort,
		Protocol:    "SRT",
		Application: harbor.Haven.Flagship.Application,
		StreamKey:   harbor.Haven.Flagship.Passphrase,
	}

	responseJson, err := response.ToJson()
	if err != nil {
		http.Error(w, "Failed to parse response to json string", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	encrypt, err := api.Encrypt(responseJson)
	if err != nil {
		http.Error(w, "Failed to encrypt response", http.StatusInternalServerError)
		return
	}

	_, err = w.Write([]byte(encrypt))
	if err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
		return
	}
}
