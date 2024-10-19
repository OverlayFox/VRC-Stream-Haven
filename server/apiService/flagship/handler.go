package flagship

import (
	"encoding/json"
	"fmt"
	"github.com/OverlayFox/VRC-Stream-Haven/apiServer"
	"github.com/OverlayFox/VRC-Stream-Haven/apiServer/escort/paths/info"
	"github.com/OverlayFox/VRC-Stream-Haven/logger"
	"github.com/OverlayFox/VRC-Stream-Haven/types"
	"io"
	"net/http"
	"time"
)

func IsApiOnline(escort *types.Escort) bool {
	url := fmt.Sprintf("http://%s:%d/escort/info", escort.BackEndIP.String(), escort.ApiPort)
	logger.HavenLogger.Debug().Msgf("Checking if Escort %s is online on port %d", escort.BackEndIP.String(), escort.ApiPort)

	client := http.Client{
		Timeout: 2 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	logger.HavenLogger.Debug().Msgf("Escort %s is currently: %d", escort.BackEndIP.String(), resp.StatusCode)

	return resp.StatusCode == http.StatusOK
}

func GetEscortReaders(escort *types.Escort) (info.Response, error) {
	url := fmt.Sprintf("http://%s:%d/escort/info", escort.BackEndIP, escort.ApiPort)

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

	bodyJson, err := apiServer.Decrypt(string(responseBody))
	if err != nil {
		return info.Response{}, err
	}

	var decodedBody info.Response
	if err := json.Unmarshal([]byte(bodyJson), &decodedBody); err != nil {
		return info.Response{}, err
	}

	return decodedBody, nil
}
