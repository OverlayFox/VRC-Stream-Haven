package apiServer

import (
	"encoding/json"
	"github.com/OverlayFox/VRC-Stream-Haven/depreciated"
	apiServer3 "github.com/OverlayFox/VRC-Stream-Haven/flagship/apiServer/register"
	apiServer2 "github.com/OverlayFox/VRC-Stream-Haven/shared/crypto"
	"github.com/OverlayFox/VRC-Stream-Haven/shared/haven"
	"github.com/OverlayFox/VRC-Stream-Haven/shared/logger"
	"github.com/gorilla/mux"
	"io"
	"net/http"
)

// RegisterEscortToHaven adds the caller as an escort to the haven
func RegisterEscortToHaven(w http.ResponseWriter, r *http.Request) {
	logger.HavenLogger.Debug().Msgf("Received escort register request from %s", r.RemoteAddr)
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	bodyJson, err := apiServer2.Decrypt(string(bodyBytes))
	if err != nil {
		http.Error(w, "Failed to decrypt body", http.StatusInternalServerError)
		return
	}
	logger.HavenLogger.Debug().Msgf("Received following body from escort: %s", bodyJson)

	var escort haven.Escort
	if err := json.Unmarshal([]byte(bodyJson), &escort); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = depreciated.Haven.RemoveEscort(escort.BackEndIP)
	if err == nil {
		logger.HavenLogger.Warn().Msgf("Escort %s already exists, removing it", escort.IpAddress)
	}

	depreciated.Haven.AddEscort(&escort)
	logger.HavenLogger.Info().Msgf("Successfully added escort %s to haven", escort.IpAddress)

	response := apiServer3.Response{
		IpAddress: depreciated.Haven.Flagship.IpAddress.String(),
		BackEndIp: depreciated.Haven.Flagship.BackEndIP.String(),
		SrtPort:   depreciated.Haven.Flagship.SrtIngestPort,
	}

	responseJson, err := response.ToJson()
	if err != nil {
		http.Error(w, "Failed to parse response to json string", http.StatusInternalServerError)
		return
	}
	logger.HavenLogger.Debug().Msgf("Sending approval response with body: %s", responseJson)

	encrypt, err := apiServer2.Encrypt(responseJson)
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

	logger.HavenLogger.Info().Msgf("Successfully send approval response to escort %s", escort.IpAddress)
}

// InitApi initializes the mux router and sets up the routes for a Flagship Endpoint.
func InitApi() *mux.Router {
	r := mux.NewRouter()
	r.Handle("/flagship/register", http.HandlerFunc(RegisterEscortToHaven)).Methods("POST")

	return r
}
