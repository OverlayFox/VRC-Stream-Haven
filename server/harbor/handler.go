package harbor

import (
	"github.com/OverlayFox/VRC-Stream-Haven/api"
	"net"
	"os"
	"strconv"

	"github.com/OverlayFox/VRC-Stream-Haven/types"
)

var Haven *types.Haven

// InitHaven is used to start up the Haven with no Escorts and one Flagship.
// The Flagship is the local server that initialised the Haven.
func InitHaven() error {
	var rtspPort int
	rtspPort, err := strconv.Atoi(os.Getenv("RTSP_PORT"))
	if err != nil || rtspPort <= 0 || rtspPort > 65535 {
		rtspPort = 554
	}

	var srtPort int
	srtPort, err = strconv.Atoi(os.Getenv("SRT_PORT"))
	if err != nil || srtPort <= 0 || srtPort > 65535 {
		rtspPort = 8554
	}

	ip, err := GetPublicIpAddress()
	if err != nil {
		return err
	}

	Haven.Flagship = &types.Flagship{
		Escort: types.Escort{
			IpAddress:      net.ParseIP(ip.IpAddress),
			RtspEgressPort: uint16(rtspPort),
			Latitude:       ip.Latitude,
			Longitude:      ip.Longitude,
		},
		SrtIngestPort: uint16(srtPort),
		Passphrase:    string(api.Key),
	}

	var escorts []*types.Escort
	Haven.Escorts = &escorts

	Haven.IsServer = true

	return nil
}

// MakeEscort creates a types.Escort with the public IP Address of the server that initialised the MakeEscort function.
func MakeEscort(rtspEgressPort uint16) (*types.Escort, error) {
	ip, err := GetPublicIpAddress()
	if err != nil {
		return &types.Escort{}, err
	}

	return &types.Escort{
		IpAddress:      net.ParseIP(ip.IpAddress),
		RtspEgressPort: rtspEgressPort,
		Latitude:       ip.Latitude,
		Longitude:      ip.Longitude,
	}, nil
}
