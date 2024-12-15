package apiServer

import (
	"encoding/json"
	"github.com/OverlayFox/VRC-Stream-Haven/depreciated"
	apiServer3 "github.com/OverlayFox/VRC-Stream-Haven/flagship/apiServer/register"
	apiServer2 "github.com/OverlayFox/VRC-Stream-Haven/shared/crypto"
	"github.com/OverlayFox/VRC-Stream-Haven/shared/logger"
	"github.com/OverlayFox/VRC-Stream-Haven/shared/overseer"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"io"
	"net/http"
)

type FlagshipApiServer struct {
	logger     zerolog.Logger
	server     *mux.Router
	passphrase []byte
	haven      *overseer.Haven
}

// NewFlagshipApiServer creates and initializes a new FlagshipApiServer
func NewFlagshipApiServer(logger zerolog.Logger, passphrase []byte, haven *overseer.Haven) *FlagshipApiServer {
	fs := &FlagshipApiServer{
		logger:     logger,
		passphrase: passphrase,
		haven:      haven,
	}

	fs.server = mux.NewRouter()
	fs.registerRoutes()

	return fs
}

// registerRoutes sets up all the HTTP routes for the server
func (fs *FlagshipApiServer) registerRoutes() {
	fs.server.HandleFunc("/flagship/register", fs.RegisterEscortToHaven).Methods("POST")
}

// Start begins listening for HTTP requests
func (fs *FlagshipApiServer) Start(address string) error {
	return http.ListenAndServe(address, fs.server)
}

// RegisterEscortToHaven adds the caller as an escort to the haven
func (fs *FlagshipApiServer) RegisterEscortToHaven(w http.ResponseWriter, r *http.Request) {
	fs.logger.Debug().Msgf("Received escort register request from %s", r.RemoteAddr)
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	bodyJson, err := apiServer2.Decrypt(string(bodyBytes), fs.passphrase)
	if err != nil {
		http.Error(w, "Failed to decrypt body", http.StatusInternalServerError)
		return
	}
	fs.logger.Debug().Msgf("Received following body from escort: %s", bodyJson)

	var escort overseer.Escort
	if err := json.Unmarshal([]byte(bodyJson), &escort); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = fs.haven.RemoveEscort(escort.BackEndIP)
	if err == nil {
		fs.logger.Warn().Msgf("Escort %s already exists, removing it", escort.IpAddress)
	}

	fs.haven.AddEscort(&escort)
	fs.logger.Info().Msgf("Successfully added escort %s to haven", escort.IpAddress)

	response := apiServer3.Response{
		IpAddress: fs.haven.Flagship.IpAddress.String(),
		BackEndIp: fs.haven.Flagship.BackEndIP.String(),
		SrtPort:   fs.haven.Flagship.SrtIngestPort,
	}

	responseJson, err := response.ToJson()
	if err != nil {
		http.Error(w, "Failed to parse response to json string", http.StatusInternalServerError)
		return
	}
	fs.logger.Debug().Msgf("Sending approval response with body: %s", responseJson)

	encrypt, err := apiServer2.Encrypt(responseJson, fs.passphrase)
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

	fs.logger.Info().Msgf("Successfully send approval response to escort %s", escort.IpAddress)
}
