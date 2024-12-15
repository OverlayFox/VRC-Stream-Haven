package apiServer

import (
	"github.com/OverlayFox/VRC-Stream-Haven/escort/apiServer/info"
	rtspEscort "github.com/OverlayFox/VRC-Stream-Haven/escort/rtspServer"
	apiServer2 "github.com/OverlayFox/VRC-Stream-Haven/shared/crypto"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"net/http"
	"os"
	"strconv"
)

type EscortApiServer struct {
	logger     zerolog.Logger
	server     *mux.Router
	passphrase []byte
}

// NewEscortApiServer creates and initializes a new EscortApiServer
func NewEscortApiServer(logger zerolog.Logger, passphrase []byte) *EscortApiServer {
	es := &EscortApiServer{
		logger:     logger,
		passphrase: passphrase,
	}

	es.server = mux.NewRouter()
	es.registerRoutes()

	return es
}

// registerRoutes sets up all the HTTP routes for the server
func (es *EscortApiServer) registerRoutes() {
	es.server.HandleFunc("/escort/info", es.GetInfo).Methods("POST")
}

// Start begins listening for HTTP requests
func (es *EscortApiServer) Start(address string) error {
	return http.ListenAndServe(address, es.server)
}

// GetInfo returns the information of the escort to the caller.
func (es *EscortApiServer) GetInfo(w http.ResponseWriter, r *http.Request) {
	currentViewers, err := rtspEscort.ServerHandler.GetReaders()
	if err != nil {
		es.logger.Error().Err(err).Msg("Could not get amount of Readers from external Stream Struct")
		http.Error(w, "Could not get amount of Readers", http.StatusInternalServerError)
		return
	}

	var maxViewers = 0
	if os.Getenv("MAX_VIEWERS") != "" && os.Getenv("MAX_VIEWERS") != "0" {
		maxViewers, err = strconv.Atoi(os.Getenv("MAX_VIEWERS"))
		if err != nil {
			maxViewers = 0
		}
	}

	response := info.Response{
		CurrentViewers:    currentViewers,
		MaxAllowedViewers: maxViewers,
	}

	responseJson, err := response.ToJson()
	if err != nil {
		es.logger.Error().Err(err).Msg("Could not parse response to json string")
		http.Error(w, "Could not parse response to json string", http.StatusInternalServerError)
		return
	}

	encrypt, err := apiServer2.Encrypt(responseJson, es.passphrase)
	if err != nil {
		es.logger.Error().Err(err).Msg("Failed to encrypt response")
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
