package flagship

import (
	"encoding/json"
	"fmt"
	"github.com/OverlayFox/VRC-Stream-Haven/api"
	"github.com/OverlayFox/VRC-Stream-Haven/api/escort/paths/info"
	"github.com/OverlayFox/VRC-Stream-Haven/types"
	"io"
	"net/http"
)

func GetEscortReaders(escort *types.Escort) (info.Response, error) {
	url := fmt.Sprintf("http://%s:%s/escort/info", escort.IpAddress, escort.ApiPort)
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return info.Response{}, err
	}
	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return info.Response{}, err
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return info.Response{}, err
	}

	bodyJson, err := api.Decrypt(string(responseBody))
	if err != nil {
		return info.Response{}, err
	}

	var decodedBody info.Response
	if err := json.Unmarshal([]byte(bodyJson), &decodedBody); err != nil {
		return info.Response{}, err
	}

	return decodedBody, nil
}
