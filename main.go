package main

import (
	lib "github.com/OverlayFox/VRC-Stream-Haven/libraries"
	"github.com/OverlayFox/VRC-Stream-Haven/servers"
	"github.com/bluenviron/gortsplib/v4"
	"log"
)

func main() {
	lib.InitialiseConfig()

	rtspHandler := &servers.RtspServerHandler{}
	rtspHandler.Server = &gortsplib.Server{
		Handler:     rtspHandler,
		RTSPAddress: ":8554",
	}

	//rtspNodeHandler := &servers.RtspNodeServerHandler{}
	//rtspNodeHandler.Server = &gortsplib.Server{
	//	Handler:     rtspNodeHandler,
	//	RTSPAddress: ":8554",
	//}

	log.Println("Server is ready.....")
	panic(rtspHandler.Server.StartAndWait())

	//log.Println("Node Server is ready.....")
	//go func() {
	//	lib.RelayHlsToRtsp("http://10.58.97.100/tmp/streams/playlist.m3u8", "rtsp://localhost:8554/ingest/channel")
	//}()
	//panic(rtspNodeHandler.Server.StartAndWait())

	//flag.Parse()
	//go servers.StartRtmpServer()
	//select {}
}
