package servers

import (
	lib "github.com/OverlayFox/VRC-Stream-Haven/libraries"
	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"github.com/bluenviron/gortsplib/v4/pkg/description"
	"github.com/bluenviron/gortsplib/v4/pkg/format"
	"github.com/pion/rtp"
	"log"
	"sync"
)

type RtspServerHandler struct {
	Server    *gortsplib.Server
	Stream    *gortsplib.ServerStream
	Publisher *gortsplib.ServerSession
	Mutex     sync.Mutex
}

func (sh *RtspServerHandler) OnConnectionOpen(ctx *gortsplib.ServerHandlerOnConnOpenCtx) {
	log.Println("Connection Opened")
}

func (sh *RtspServerHandler) OnConnectionClose(ctx *gortsplib.ServerHandlerOnConnCloseCtx) {
	log.Println("Connection Closed")
}

func (sh *RtspServerHandler) OnSessionOpen(ctx *gortsplib.ServerHandlerOnSessionOpenCtx) {
	log.Println("Session opened")
}

func (sh *RtspServerHandler) OnSessionClose(ctx *gortsplib.ServerHandlerOnSessionCloseCtx) {
	log.Println("Session closed")

	sh.Mutex.Lock()
	defer sh.Mutex.Unlock()

	if sh.Stream != nil && ctx.Session == sh.Publisher {
		sh.Stream.Close()
		sh.Stream = nil
	}
}

func (sh *RtspServerHandler) OnDescribe(ctx *gortsplib.ServerHandlerOnDescribeCtx) (*base.Response, *gortsplib.ServerStream, error) {
	log.Println("Describe Request")

	sh.Mutex.Lock()
	defer sh.Mutex.Unlock()

	if sh.Stream == nil {
		return &base.Response{
			StatusCode: base.StatusBadRequest,
		}, nil, nil
	}

	latitude, longitude := lib.LocateIp(ctx.Conn.NetConn().RemoteAddr().String())
	closestNode := lib.GetDistance(latitude, longitude, lib.Config.Nodes)
	log.Printf("Client IP-Address: %v", ctx.Conn.NetConn().RemoteAddr().String())

	if closestNode.IpAddress != lib.Config.Server.IpAddress {
		log.Printf("Send Client to node: %v", closestNode.IpAddress)
		return &base.Response{
			StatusCode: base.StatusMovedPermanently,
			Header: base.Header{
				"Location": base.HeaderValue{"rtsp://" + closestNode.IpAddress + ":" + closestNode.StreamingPort},
			},
		}, nil, nil
	} else {
		log.Printf("Send Client to server: %v", lib.Config.Server.IpAddress)
	}

	return &base.Response{
		StatusCode: base.StatusOK,
	}, sh.Stream, nil

}

func (sh *RtspServerHandler) OnAnnounce(ctx *gortsplib.ServerHandlerOnAnnounceCtx) (*base.Response, error) {
	log.Println("Announce Request")

	sh.Mutex.Lock()
	defer sh.Mutex.Unlock()

	if sh.Stream != nil {
		sh.Stream.Close()
		sh.Publisher.Close()
	}

	sh.Stream = gortsplib.NewServerStream(sh.Server, ctx.Description)
	sh.Publisher = ctx.Session

	return &base.Response{
		StatusCode: base.StatusOK,
	}, nil
}

func (sh *RtspServerHandler) OnSetup(ctx *gortsplib.ServerHandlerOnSetupCtx) (*base.Response, *gortsplib.ServerStream, error) {
	log.Println("Setup Request")

	if sh.Stream == nil {
		return &base.Response{
			StatusCode: base.StatusNotFound,
		}, nil, nil
	}

	return &base.Response{
		StatusCode: base.StatusOK,
	}, sh.Stream, nil
}

func (sh *RtspServerHandler) OnRecord(ctx *gortsplib.ServerHandlerOnRecordCtx) (*base.Response, error) {
	log.Println("Record Request")

	ctx.Session.OnPacketRTPAny(func(media *description.Media, format format.Format, packet *rtp.Packet) {
		sh.Stream.WritePacketRTP(media, packet)
	})

	go func() {
		lib.NodeHlsPlaylist("rtsp://127.0.0.1:8554")
	}()

	return &base.Response{
		StatusCode: base.StatusOK,
	}, nil
}

func (sh *RtspServerHandler) OnPlay(ctx *gortsplib.ServerHandlerOnPlayCtx) (*base.Response, error) {
	log.Println("Play Request")

	if sh.Stream != nil {
		return &base.Response{
			StatusCode: base.StatusOK,
		}, nil
	}

	return &base.Response{
		StatusCode: base.StatusNotFound,
	}, nil
}
