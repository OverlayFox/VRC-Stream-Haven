package haven

import (
	"encoding/json"
	"github.com/OverlayFox/VRC-Stream-Haven/haven/types/responses"
	"github.com/OverlayFox/VRC-Stream-Haven/logger"
	"math/rand"
	"net/http"
	"time"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// GenerateKey Generates a random key which can be used as a StreamKey or Passphrase.
func GenerateKey() string {
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	key := make([]byte, 32)
	for i := range key {
		key[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(key)
}

// GetPublicIpAddress Gets the Public IP Address of the Server.
func GetPublicIpAddress() (responses.IpApi, error) {
	response, err := http.Get("http://ip-api.com/json")
	if err != nil {
		logger.Log.Error().Err(err).Msg("IP-API is not reachable.")
		return responses.IpApi{}, err
	}

	var body responses.IpApi
	err = json.NewDecoder(response.Body).Decode(&body)
	if err != nil {
		logger.Log.Error().Err(err).Msg("Could not decode IP-API response.")
		return responses.IpApi{}, err
	}

	return body, nil
}
