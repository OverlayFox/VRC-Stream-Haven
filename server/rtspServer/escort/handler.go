package flagship

import (
	"github.com/bluenviron/gortsplib/v4"
	"strconv"
	"time"
)

var ServerHandler *EscortHandler

func InitRtspServer(rtspPortInt int) *EscortHandler {
	rtspPort := ":" + strconv.Itoa(rtspPortInt)

	ServerHandler := &EscortHandler{}
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
