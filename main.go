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

	log.Println("Server is ready.....")
	panic(rtspHandler.Server.StartAndWait())

	//flag.Parse()
	//go servers.StartRtmpServer()
	//select {}
}
