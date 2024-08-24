package main

import (
	"github.com/OverlayFox/VRC-Stream-Haven/logger"
	"github.com/OverlayFox/VRC-Stream-Haven/servers/upnp"
)

func asFlagship() {

}

func main() {
	logger.InitLogger()

	upnp.SetupPortForward(9710, 8557, 8042)

	logger.Log.Info().Msg("UPnP setup completed...")

	//ip, err := servers.GetLocalIP()
	//if err != nil {
	//	logger.Log.Fatal().Err(err).Msg("Could not get local IP")
	//}
	//
	//portMappings := []types.PortMapping{{
	//	ExternalPort: 42,
	//	Protocol:     "UDP",
	//	InternalPort: 42,
	//	InternalIP:   ip.String(),
	//	Enabled:      true,
	//	Description:  "VRC Haven Test",
	//}}
	//
	//err = upnp.ForwardPorts(portMappings)
	//if err != nil {
	//	logger.Log.Fatal().Err(err).Msg("Could not forward ports")
	//}

	//escort, err := haven.MakeEscort(8554)
	//if err != nil {
	//	logger.Log.Fatal().Err(err).Msg("Could not create Escort Flagship")
	//}
	//flagship := haven.MakeFlagship(escort, 8080, 8555, 1935)
	//haven.MakeHaven(&[]*types.Escort{escort}, flagship, true)
	//
	//go func() {
	//	rtmp.StartRtmpServer()
	//}()
	//
	//logger.Log.Info().Msg("RTMP Server started")
	//select {}

	//lib.Scanner = bufio.NewScanner(os.Stdin)
	//if lib.IsServer() {
	//	if lib.IsIngestTypeSrt() {
	//		ingestSrtServer := servers.SetupIngestSrt(lib.GetSrtIngestPort())
	//	} else {
	//		ingestRtmpServer := servers.SetupIngestRtmp(lib.GetRtmpIngestPort())
	//	}
	//
	//	Config.Server.RtspEgressPort = lib.GetRtspEgressPort(true)
	//	Config.Server.IpAddress = lib.GetPublicIpAddress()
	//	Config.Nodes = append(Config.Nodes, lib.GetNodes()...)
	//	Config.Server.Passphrase = lib.GenerateKey()
	//	backendSrtServer := servers.SetupBackendSrt(Config.Server.Passphrase)
	//
	//} else {
	//
	//}

	//servers.StartUpIngestSRT()

	//lib.GeoDatabase = lib.LoadDatabase()
	//lib.InitialiseConfig()

	//rtspHandler := &servers.RtspServerHandler{}
	//rtspHandler.Server = &gortsplib.Server{
	//	Handler:     rtspHandler,
	//	RTSPAddress: lib.Config.Server.RtspStreamingPortString(),
	//}
	//go rtspHandler.Server.StartAndWait()
	//
	//if os.Getenv("IS_NODE") == "False" {
	//	go servers.StartRtmpServer()
	//}
	//
	//select {}
}
