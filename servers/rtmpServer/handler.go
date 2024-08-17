package rtmpServer

import (
	"github.com/OverlayFox/VRC-Stream-Haven/haven"
	"github.com/OverlayFox/VRC-Stream-Haven/servers/rtmpServer/types"
	"net"
	"strconv"
)

func StartRtmpServer() {
	addr := "127.0.0.1:" + strconv.Itoa(haven.Haven.Flagship.RtmpIngestPort)
	listen, _ := net.Listen("tcp4", addr)

	var sess *types.MediaSession
	for {
		conn, _ := listen.Accept()
		sess = types.NewMediaSession(conn)
		sess.Init()
		go sess.Start()
	}
}
