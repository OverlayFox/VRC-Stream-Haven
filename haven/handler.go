package haven

import (
	"github.com/OverlayFox/VRC-Stream-Haven/haven/types"
	"net"
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

func MakeFlagship(ship *types.Escort, apiPort, srtIngestPort, rtmpIngestPort uint16) *types.Flagship {
	return &types.Flagship{
		Ship:           ship,
		SrtIngestPort:  srtIngestPort,
		RtmpIngestPort: rtmpIngestPort,
		ApiPort:        apiPort,
		Passphrase:     GenerateKey(),
	}
}
