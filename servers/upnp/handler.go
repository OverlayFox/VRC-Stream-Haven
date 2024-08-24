package upnp

import (
	"context"
	"errors"
	"github.com/OverlayFox/VRC-Stream-Haven/logger"
	"github.com/OverlayFox/VRC-Stream-Haven/servers"
	"github.com/OverlayFox/VRC-Stream-Haven/servers/upnp/types"
	"github.com/OverlayFox/VRC-Stream-Haven/ui"
	"github.com/huin/goupnp/dcps/internetgateway2"
	"golang.org/x/sync/errgroup"
)

func forwardPorts(portMappings []types.PortMapping, client types.RouterClient) error {
	localIp, err := servers.GetLocalIP()
	if err != nil {
		return err
	}

	for _, portMapping := range portMappings {
		err = client.AddPortMapping(
			"",
			portMapping.ExternalPort,
			portMapping.Protocol,
			portMapping.InternalPort,
			localIp.String(),
			portMapping.Enabled,
			portMapping.Description,
			0)

		if err != nil {
			return err
		}
	}

	return nil
}

func getRouterClient(ctx context.Context) (types.RouterClient, error) {
	tasks, _ := errgroup.WithContext(ctx)

	var ip1Clients []*internetgateway2.WANIPConnection1
	tasks.Go(func() error {
		var err error
		ip1Clients, _, err = internetgateway2.NewWANIPConnection1Clients()
		return err
	})
	var ip2Clients []*internetgateway2.WANIPConnection2
	tasks.Go(func() error {
		var err error
		ip2Clients, _, err = internetgateway2.NewWANIPConnection2Clients()
		return err
	})
	var ppp1Clients []*internetgateway2.WANPPPConnection1
	tasks.Go(func() error {
		var err error
		ppp1Clients, _, err = internetgateway2.NewWANPPPConnection1Clients()
		return err
	})

	if err := tasks.Wait(); err != nil {
		return nil, err
	}

	switch {
	case len(ip2Clients) == 1:
		return ip2Clients[0], nil
	case len(ip1Clients) == 1:
		return ip1Clients[0], nil
	case len(ppp1Clients) == 1:
		return ppp1Clients[0], nil
	default:
		return nil, errors.New("multiple or no services found")
	}
}

func SetupPortForward(srtPort, rtspPort, apiPort uint16) {
	router, err := getRouterClient(context.Background())
	if err != nil {
		switch ui.PortForwardNotPossible() {
		case ui.IDRETRY:
			SetupPortForward(srtPort, rtspPort, apiPort)

		case ui.IDIGNORE:
			logger.Log.Info().Msg("Ignoring UPnP port forwarding...")
			return

		default:
			logger.Log.Fatal().Msg("Exiting upon user request...")
			return
		}
	}

	localIp, err := servers.GetLocalIP()
	if err != nil {
		logger.Log.Fatal().Err(err).Msg("Could not get local IP")
	}

	var ports []types.PortMapping
	if srtPort != 0 {
		ports = append(ports, types.PortMapping{
			ExternalPort: srtPort,
			Protocol:     "UDP",
			InternalPort: srtPort,
			InternalIP:   localIp.String(),
			Enabled:      true,
			Description:  "VRC-Haven SRT Forward",
		})
	}

	if rtspPort != 0 {
		ports = append(ports, types.PortMapping{
			ExternalPort: rtspPort,
			Protocol:     "UDP",
			InternalPort: rtspPort,
			InternalIP:   localIp.String(),
			Enabled:      true,
			Description:  "VRC-Haven RTSP Forward",
		})
	}

	if apiPort != 0 {
		ports = append(ports, types.PortMapping{
			ExternalPort: apiPort,
			Protocol:     "TCP",
			InternalPort: apiPort,
			InternalIP:   localIp.String(),
			Enabled:      true,
			Description:  "VRC-Haven API Forward",
		})
	}

	err = forwardPorts(ports, router)
	if err != nil {
		logger.Log.Fatal().Err(err).Msg("Could not forward ports")
	}
}
