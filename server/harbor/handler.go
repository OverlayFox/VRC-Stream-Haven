package harbor

import (
	"net"
	"os"

	"github.com/OverlayFox/VRC-Stream-Haven/types"
)

var Haven *types.Haven

// InitHaven is used to start up the Haven with no Escorts and one Flagship.
// The Flagship is the local server that initialised the Haven.
func InitHaven(srtIngestPort, rtspEgressPort uint16) error {
	ip, err := GetPublicIpAddress()
	if err != nil {
		return err
	}

	Haven.Flagship = &types.Flagship{
		Escort: types.Escort{
			IpAddress:      net.ParseIP(ip.IpAddress),
			RtspEgressPort: rtspEgressPort,
			Latitude:       ip.Latitude,
			Longitude:      ip.Longitude,
		},
		SrtIngestPort: srtIngestPort,
		Passphrase:    os.Getenv("PASSPHRASE"),
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
