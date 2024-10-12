package harbor

import (
	"net"

	"github.com/OverlayFox/VRC-Stream-Haven/types"
)

var Haven *types.Haven

func MakeHaven(escorts *[]*types.Escort, flagship *types.Flagship, isServer bool) {
	Haven.Escorts = escorts
	Haven.Flagship = flagship
	Haven.IsServer = isServer
}

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

func MakeFlagship(ship *types.Escort, srtIngestPort uint16, application string) *types.Flagship {
	return &types.Flagship{
		Ship:          ship,
		SrtIngestPort: srtIngestPort,
		Application:   application,
		Passphrase:    GenerateKey(),
	}
}
