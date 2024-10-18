package harbor

import (
	"github.com/OverlayFox/VRC-Stream-Haven/types"
)

var Haven *types.Haven

// MakeHaven is used to start up the Haven with one Escorts and one Flagship.
// The Flagship is the local server that initialised the Haven.
func MakeHaven(escort types.Escort, srtPort uint16, passpharse string) *types.Haven {
	Haven = &types.Haven{}

	Haven.Flagship = &types.Flagship{
		Escort:        escort,
		SrtIngestPort: srtPort,
		Passphrase:    passpharse,
	}

	var escorts []*types.Escort
	Haven.Escorts = &escorts
	*Haven.Escorts = append(*Haven.Escorts, &escort)

	return Haven
}

// MakeEscort creates a types.Escort with the public IP Address of the server that initialised the MakeEscort function.
func MakeEscort(rtspEgressPort, apiPort uint16) (*types.Escort, error) {
	ip, err := GetPublicIpAddress()
	if err != nil {
		return &types.Escort{}, err
	}

	return &types.Escort{
		IpAddress:      ip.IpAddress,
		RtspEgressPort: rtspEgressPort,
		Latitude:       ip.Latitude,
		Longitude:      ip.Longitude,
		ApiPort:        apiPort,
	}, nil
}
