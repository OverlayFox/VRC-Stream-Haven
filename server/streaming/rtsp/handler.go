package rtsp

import (
	rtspServer "github.com/OverlayFox/VRC-Stream-Haven/streaming/rtsp/types"
	"github.com/bluenviron/gortsplib/v4"
	"os"
	"strconv"
	"time"
)

var ServerHandler *rtspServer.RtspHandler

func init() {
	rtspPortInt, err := strconv.Atoi(os.Getenv("RTSP_PORT"))
	if err != nil || rtspPortInt <= 0 || rtspPortInt > 65535 {
		rtspPortInt = 554
	}
	rtspPort := ":" + strconv.Itoa(rtspPortInt)

	ServerHandler = &rtspServer.RtspHandler{}

	ServerHandler.Server = &gortsplib.Server{
		RTSPAddress:              rtspPort,
		ReadTimeout:              10 * time.Second,
		WriteTimeout:             10 * time.Second,
		WriteQueueSize:           512,
		MaxPacketSize:            1472,
		DisableRTCPSenderReports: false,
		Handler:                  ServerHandler,
	}
}
