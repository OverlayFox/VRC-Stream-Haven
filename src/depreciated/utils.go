package depreciated

import (
	"encoding/json"
	"github.com/OverlayFox/VRC-Stream-Haven/shared/logger"
	"math/rand"
	"net"
	"net/http"
	"time"

	"github.com/OverlayFox/VRC-Stream-Haven/types"
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
func GetPublicIpAddress() (types.IpApi, error) {
	response, err := http.Get("http://ip-api.com/json")
	if err != nil {
		logger.HavenLogger.Error().Err(err).Msg("IP-API is not reachable.")
		return types.IpApi{}, err
	}

	var body types.IpApi
	err = json.NewDecoder(response.Body).Decode(&body)
	if err != nil {
		logger.HavenLogger.Error().Err(err).Msg("Could not decode IP-API response.")
		return types.IpApi{}, err
	}

	return body, nil
}

func IsPortFree(port string) bool {
	listener, err := net.Listen("udp", ":"+port)
	if err != nil {
		// Port is not free
		return false
	}

	listener.Close()
	return true
}

func GetLocalIP() (net.IP, error) {
	conn, err := net.Dial("udp", "1.1.1.1:80")
	if err != nil {
		return net.IP{}, err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP, nil
}
