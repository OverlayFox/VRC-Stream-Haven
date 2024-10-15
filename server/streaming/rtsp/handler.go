package rtsp

import (
	rtspServer "github.com/OverlayFox/VRC-Stream-Haven/types"
	"github.com/bluenviron/gortsplib/v4"
	"strconv"
	"time"
)

var ServerHandler *rtspServer.RtspHandler

func InitRtspServer(rtspPortInt int) *rtspServer.RtspHandler {
	rtspPort := ":" + strconv.Itoa(rtspPortInt)

	ServerHandler := &rtspServer.RtspHandler{}
	ServerHandler.Server = &gortsplib.Server{
		RTSPAddress:              rtspPort,
		ReadTimeout:              10 * time.Second,
		WriteTimeout:             10 * time.Second,
		WriteQueueSize:           512,
		MaxPacketSize:            1472,
		DisableRTCPSenderReports: false,
		Handler:                  ServerHandler,
	}

	return ServerHandler
}
