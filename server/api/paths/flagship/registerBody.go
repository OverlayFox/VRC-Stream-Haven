package flagship

import (
	"encoding/json"
	"github.com/OverlayFox/VRC-Stream-Haven/types"
	"math/rand"
	"os"
	"strconv"
)

type RegisterBody struct {
	IpAddress      string  `yaml:"ipAddress"`
	RtspEgressPort uint16  `yaml:"rtspEgressPort"`
	Latitude       float64 `yaml:"lat"`
	Longitude      float64 `yaml:"lon"`
	Username       string  `yaml:"username"`
}

func (r *RegisterBody) ToJson() (string, error) {
	jsonData, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}

func BuildBody(escort *types.Escort) RegisterBody {
	var hostname string

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unkown-hostname-" + strconv.Itoa(rand.Intn(1001))
	}

	return RegisterBody{
		IpAddress:      escort.IpAddress.String(),
		RtspEgressPort: escort.RtspEgressPort,
		Latitude:       escort.Latitude,
		Longitude:      escort.Longitude,
		Username:       hostname,
	}
}
