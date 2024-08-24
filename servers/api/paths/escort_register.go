package paths

import (
	"encoding/json"
	"github.com/OverlayFox/VRC-Stream-Haven/haven"
	"github.com/OverlayFox/VRC-Stream-Haven/haven/types"
	"github.com/OverlayFox/VRC-Stream-Haven/servers/api"
	"io"
	"net"
	"net/http"
)

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	bodyJson, err := api.Decrypt(string(bodyBytes), api.PrePassphrase)
	if err != nil {
		http.Error(w, "Failed to decrypt body", http.StatusInternalServerError)
		return
	}

	var body RegisterBody
	if err := json.Unmarshal([]byte(bodyJson), &body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	//remove escort if it already exists
	escorts := *haven.Haven.Escorts
	for i := 0; i < len(escorts); {
		if escorts[i].IpAddress.Equal(net.ParseIP(body.IpAddress)) {
			escorts = append(escorts[:i], escorts[i+1:]...)
		} else {
			i++
		}
	}
	haven.Haven.Escorts = &escorts

	*haven.Haven.Escorts = append(*haven.Haven.Escorts, &types.Escort{
		IpAddress:      net.ParseIP(body.IpAddress),
		RtspEgressPort: body.RtspEgressPort,
		Latitude:       body.Latitude,
		Longitude:      body.Longitude,
		Username:       body.Username,
		Passphrase:     body.Passphrase,
	})

	w.WriteHeader(http.StatusOK)
	encrypt, err := api.Encrypt("Successfully registered escort", []byte(body.Passphrase))
	if err != nil {
		http.Error(w, "Failed to encrypt response", http.StatusInternalServerError)
		return
	}

	_, err = w.Write([]byte(encrypt))
	if err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
		return
	}
}
