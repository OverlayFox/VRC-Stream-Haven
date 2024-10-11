package haven

import (
	types2 "github.com/OverlayFox/VRC-Stream-Haven/server/haven/types"
	"net"
)

var Haven *types2.Haven

func MakeHaven(escorts *[]*types2.Escort, flagship *types2.Flagship, isServer bool) {
	Haven.Escorts = escorts
	Haven.Flagship = flagship
	Haven.IsServer = isServer
}

func MakeEscort(rtspEgressPort uint16) (*types2.Escort, error) {
	ip, err := GetPublicIpAddress()
	if err != nil {
		return &types2.Escort{}, err
	}

	return &types2.Escort{
		IpAddress:      net.ParseIP(ip.IpAddress),
		RtspEgressPort: rtspEgressPort,
		Latitude:       ip.Latitude,
		Longitude:      ip.Longitude,
	}, nil
}

func MakeFlagship(ship *types2.Escort, apiPort, srtIngestPort, rtmpIngestPort uint16) *types2.Flagship {
	return &types2.Flagship{
		Ship:           ship,
		SrtIngestPort:  srtIngestPort,
		RtmpIngestPort: rtmpIngestPort,
		ApiPort:        apiPort,
		Passphrase:     GenerateKey(),
	}
}
