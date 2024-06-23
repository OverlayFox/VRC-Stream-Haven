package main

import (
	lib "github.com/OverlayFox/VRC-Stream-Haven/libraries"
	"github.com/OverlayFox/VRC-Stream-Haven/servers"
	"github.com/bluenviron/gortsplib/v4"
	"os"
)

func main() {
	lib.InitialiseConfig()
	lib.InitEnv()

	rtspHandler := &servers.RtspServerHandler{}
	rtspHandler.Server = &gortsplib.Server{
		Handler:     rtspHandler,
		RTSPAddress: ":8554",
	}
	go rtspHandler.Server.StartAndWait()

	if os.Getenv("IS_NODE") == "False" {
		go servers.StartRtmpServer()
	}
	select {}
}
