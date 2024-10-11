package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/OverlayFox/VRC-Stream-Haven/api/paths/flagship"
	"github.com/OverlayFox/VRC-Stream-Haven/harbor"
	"github.com/OverlayFox/VRC-Stream-Haven/types"
	"io"
	"net"
	"net/http"
	"os"
)

var url = os.Getenv("API_URL")

func RegisterEscort(escort *types.Escort) error {
	if url == "" {
		return fmt.Errorf("API_URL not set")
	}

	regBody := flagship.BuildBody(escort)
	body, err := regBody.ToJson()
	if err != nil {
		return err
	}
	encryptedBody, err := Encrypt(body)

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

	bodyJson, err := Decrypt(string(responseBody))
	if err != nil {
		return err
	}

	var decodedBody flagship.RegisterResponse
	if err := json.Unmarshal([]byte(bodyJson), &decodedBody); err != nil {
		return err
	}

	harbor.Haven.Flagship = &types.Flagship{
		Ship: &types.Escort{
			IpAddress:      net.ParseIP(decodedBody.IpAddress),
			RtspEgressPort: 0,
			Latitude:       0,
			Longitude:      0,
			Username:       "",
			Passphrase:     "",
		},
		SrtIngestPort: decodedBody.Port,
		Application:   decodedBody.Application,
		Passphrase:    decodedBody.StreamKey,
	}

	return nil
}
