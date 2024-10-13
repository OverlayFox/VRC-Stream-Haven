package escort

import (
	"bytes"
	"encoding/json"
	"github.com/OverlayFox/VRC-Stream-Haven/api"
	"github.com/OverlayFox/VRC-Stream-Haven/api/flagship/paths/register"
	"github.com/OverlayFox/VRC-Stream-Haven/logger"
	"io"
	"net"
	"net/http"
	"os"

	"github.com/OverlayFox/VRC-Stream-Haven/harbor"
	"github.com/OverlayFox/VRC-Stream-Haven/types"
)

var url = os.Getenv("API_URL")

// RegisterEscortWithHaven adds the current escort to the haven via an API call
func RegisterEscortWithHaven(escort *types.Escort) error {
	if url == "" {
		logger.HavenLogger.Fatal().Msg("API_URL is not set")
	}

	harbor.Haven.IsServer = false

	body, err := escort.ToJson()
	if err != nil {
		return err
	}
	encryptedBody, err := api.Encrypt(body)

	request, err := http.NewRequest("GET", url+"/flagship/register", bytes.NewBufferString(encryptedBody))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}

	bodyJson, err := api.Decrypt(string(responseBody))
	if err != nil {
		return err
	}

	var decodedBody register.Response
	if err := json.Unmarshal([]byte(bodyJson), &decodedBody); err != nil {
		return err
	}

	harbor.Haven.Flagship = &types.Flagship{
		Escort: types.Escort{
			IpAddress:      net.ParseIP(decodedBody.IpAddress),
			RtspEgressPort: 0,
			Latitude:       0,
			Longitude:      0,
		},
		SrtIngestPort: decodedBody.SrtPort,
	}
	harbor.Haven.IsServer = false

	return nil
}
