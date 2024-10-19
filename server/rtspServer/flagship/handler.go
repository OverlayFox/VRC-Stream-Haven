package flagship

import (
	"github.com/bluenviron/gortsplib/v4"
	"strconv"
	"time"
)

var ServerHandler *FlagshipHandler

func InitRtspServer(rtspPortInt int, streamkey string) *FlagshipHandler {
	rtspPort := ":" + strconv.Itoa(rtspPortInt)

	ServerHandler := &FlagshipHandler{}
	ServerHandler.Server = &gortsplib.Server{
		RTSPAddress:              rtspPort,
		ReadTimeout:              10 * time.Second,
		WriteTimeout:             10 * time.Second,
		WriteQueueSize:           512,
		MaxPacketSize:            1472,
		DisableRTCPSenderReports: false,
		Handler:                  ServerHandler,
	}
	ServerHandler.Streamkey = streamkey

	return ServerHandler
}
