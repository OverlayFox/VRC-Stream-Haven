package apiServer

import (
	"encoding/json"
	"github.com/OverlayFox/VRC-Stream-Haven/apiServer/escort/paths/info"
	"github.com/OverlayFox/VRC-Stream-Haven/apiServer/flagship/paths/register"
	"github.com/OverlayFox/VRC-Stream-Haven/harbor"
	"github.com/OverlayFox/VRC-Stream-Haven/logger"
	rtspEscort "github.com/OverlayFox/VRC-Stream-Haven/rtspServer/escort"
	"github.com/OverlayFox/VRC-Stream-Haven/types"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"io"
	"net/http"
	"os"
	"strconv"
)

// GetInfo returns the information of the escort to the caller.
func GetInfo(w http.ResponseWriter, r *http.Request) {
	currentViewers, err := rtspEscort.ServerHandler.GetReaders()
	if err != nil {
		logger.HavenLogger.Error().Err(err).Msg("Could not get amount of Readers from external Stream Struct")
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
		logger.HavenLogger.Error().Err(err).Msg("Could not parse response to json string")
		http.Error(w, "Could not parse response to json string", http.StatusInternalServerError)
		return
	}

	encrypt, err := Encrypt(responseJson)
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

// RegisterEscortToHaven adds the caller as an escort to the haven
func RegisterEscortToHaven(w http.ResponseWriter, r *http.Request) {
	logger.HavenLogger.Debug().Msgf("Received escort register request from %s", r.RemoteAddr)
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	bodyJson, err := Decrypt(string(bodyBytes))
	if err != nil {
		http.Error(w, "Failed to decrypt body", http.StatusInternalServerError)
		return
	}
	logger.HavenLogger.Debug().Msgf("Received following body from escort: %s", bodyJson)

	var escort types.Escort
	if err := json.Unmarshal([]byte(bodyJson), &escort); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = harbor.Haven.RemoveEscort(escort.BackEndIP)
	if err == nil {
		logger.HavenLogger.Warn().Msgf("Escort %s already exists, removing it", escort.IpAddress)
	}

	harbor.Haven.AddEscort(&escort)
	logger.HavenLogger.Info().Msgf("Successfully added escort %s to haven", escort.IpAddress)

	response := register.Response{
		IpAddress: harbor.Haven.Flagship.IpAddress.String(),
		BackEndIp: harbor.Haven.Flagship.BackEndIP.String(),
		SrtPort:   harbor.Haven.Flagship.SrtIngestPort,
	}

	responseJson, err := response.ToJson()
	if err != nil {
		http.Error(w, "Failed to parse response to json string", http.StatusInternalServerError)
		return
	}
	logger.HavenLogger.Debug().Msgf("Sending approval response with body: %s", responseJson)

	encrypt, err := Encrypt(responseJson)
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

// jwtMiddleware checks for a valid JWT token in the Authorization header
func jwtMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			http.Error(w, "Token is missing", http.StatusUnauthorized)
			return
		}

		tokenString = tokenString[len("Bearer "):]
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return Key, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Invalid Token", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// InitFlagshipApi initializes the mux router and sets up the routes for a Flagship Endpoint.
func InitFlagshipApi() *mux.Router {
	r := mux.NewRouter()
	r.Handle("/flagship/register", http.HandlerFunc(RegisterEscortToHaven)).Methods("POST")

	return r
}

// InitEscortApi initializes the mux router and sets up the routes for a Escort Endpoint.
func InitEscortApi() *mux.Router {
	r := mux.NewRouter()
	r.Handle("/escort/info", http.HandlerFunc(GetInfo)).Methods("GET")

	return r
}
