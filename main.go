package main

import (
	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"github.com/bluenviron/gortsplib/v4/pkg/description"
	"github.com/bluenviron/gortsplib/v4/pkg/format"
	"github.com/pion/rtp"
	"log"
	"sync"
)

type serverHandler struct {
	server    *gortsplib.Server
	stream    *gortsplib.ServerStream
	publisher *gortsplib.ServerSession
	mutex     sync.Mutex
}

func (sh *serverHandler) OnConnectionOpen(ctx *gortsplib.ServerHandlerOnConnOpenCtx) {
	log.Println("Connection Opened")
}

func (sh *serverHandler) OnConnectionClose(ctx *gortsplib.ServerHandlerOnConnCloseCtx) {
	log.Println("Connection Closed")
}

func (sh *serverHandler) OnSessionOpen(ctx *gortsplib.ServerHandlerOnSessionOpenCtx) {
	log.Println("Session opened")
}

func (sh *serverHandler) OnSessionClose(ctx *gortsplib.ServerHandlerOnSessionCloseCtx) {
	log.Println("Session closed")

	sh.mutex.Lock()
	defer sh.mutex.Unlock()

	if sh.stream != nil && ctx.Session == sh.publisher {
		sh.stream.Close()
		sh.stream = nil
	}
}

func (sh *serverHandler) OnDescribe(ctx *gortsplib.ServerHandlerOnDescribeCtx) (*base.Response, *gortsplib.ServerStream, error) {
	log.Println("Describe Request")

	sh.mutex.Lock()
	defer sh.mutex.Unlock()

	if sh.stream == nil {
		return &base.Response{
			StatusCode: base.StatusNotFound,
		}, nil, nil
	}

	return &base.Response{
		StatusCode: base.StatusOK,
	}, sh.stream, nil
}

func (sh *serverHandler) OnAnnounce(ctx *gortsplib.ServerHandlerOnAnnounceCtx) (*base.Response, error) {
	log.Println("Announce Request")

	sh.mutex.Lock()
	defer sh.mutex.Unlock()

	if sh.stream != nil {
		sh.stream.Close()
		sh.publisher.Close()
	}

	sh.stream = gortsplib.NewServerStream(sh.server, ctx.Description)
	sh.publisher = ctx.Session

	return &base.Response{
		StatusCode: base.StatusOK,
	}, nil
}

func (sh *serverHandler) OnSetup(ctx *gortsplib.ServerHandlerOnSetupCtx) (*base.Response, *gortsplib.ServerStream, error) {
	log.Println("Setup Request")

	if sh.stream == nil {
		return &base.Response{
			StatusCode: base.StatusNotFound,
		}, nil, nil
	}

	return &base.Response{
		StatusCode: base.StatusOK,
	}, sh.stream, nil
}

func (sh *serverHandler) OnRecord(ctx *gortsplib.ServerHandlerOnRecordCtx) (*base.Response, error) {
	log.Println("Record Request")

	ctx.Session.OnPacketRTPAny(func(media *description.Media, format format.Format, packet *rtp.Packet) {
		sh.stream.WritePacketRTP(media, packet)
		log.Println(format.Codec(), format.PTSEqualsDTS(packet))
	})

	return &base.Response{
		StatusCode: base.StatusOK,
	}, nil
}

func main() {
	handler := &serverHandler{}
	handler.server = &gortsplib.Server{

		Handler:     handler,
		RTSPAddress: ":8554",
	}

	log.Println("Server is ready.....")
	panic(handler.server.StartAndWait())
}
