package escort

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/OverlayFox/VRC-Stream-Haven/apiServer"
	"github.com/OverlayFox/VRC-Stream-Haven/apiServer/flagship/paths/register"
	"github.com/OverlayFox/VRC-Stream-Haven/harbor"
	"github.com/OverlayFox/VRC-Stream-Haven/logger"
	"github.com/OverlayFox/VRC-Stream-Haven/types"
	"io"
	"net"
	"net/http"
)

// RegisterEscortWithHaven adds the current escort to the haven via an API call
func RegisterEscortWithHaven(escort *types.Escort, flagshipIp net.IP, flagshipApiPort int, passphrase []byte) error {
	logger.HavenLogger.Debug().Msg("Registering Escort with Flagship")
	url := fmt.Sprintf("http://%s:%d", flagshipIp.String(), flagshipApiPort)

	jsonData, err := json.Marshal(escort)
	if err != nil {
		return err
	}
	logger.HavenLogger.Debug().Msgf("Body used to register with Flagship: %s", string(jsonData))
	encryptedBody, err := apiServer.Encrypt(string(jsonData))

	request, err := http.NewRequest("POST", url+"/flagship/register", bytes.NewBufferString(encryptedBody))
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

	bodyJson, err := apiServer.Decrypt(string(responseBody))
	if err != nil {
		return err
	}

	var decodedBody register.Response
	if err := json.Unmarshal([]byte(bodyJson), &decodedBody); err != nil {
		return err
	}

	flagshipEscort := types.Escort{
		IpAddress:      net.ParseIP(decodedBody.IpAddress),
		BackEndIP:      net.ParseIP(decodedBody.BackEndIp),
		RtspEgressPort: 0,
		Latitude:       0,
		Longitude:      0,
	}
	harbor.MakeHaven(flagshipEscort, decodedBody.SrtPort, string(passphrase))
	logger.HavenLogger.Info().Msgf("Successfully registered Escort with Flagship at %s", harbor.Haven.Flagship.IpAddress.String())

	return nil
}
