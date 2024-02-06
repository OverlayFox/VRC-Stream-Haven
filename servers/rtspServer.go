package servers

import (
	"sync"

	"github.com/OverlayFox/VRC-Stream-Haven/libraries"
	"github.com/OverlayFox/VRC-Stream-Haven/logger"
	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"github.com/bluenviron/gortsplib/v4/pkg/description"
)

type RtspServer struct {
	server *gortsplib.Server
	mutex  sync.Mutex
	stream *gortsplib.ServerStream
}

var log = logger.Logger

func (server *RtspServer) OnDescribe(ctx *gortsplib.ServerHandlerOnDescribeCtx) (*base.Response, *gortsplib.ServerStream, error) {
	log.Printf("Describe Request")

	server.mutex.Lock()
	defer server.mutex.Unlock()

	if server.stream == nil {
		return &base.Response{
			StatusCode: base.StatusNotFound,
		}, nil, nil
	}

	return &base.Response{
		StatusCode: base.StatusOK,
	}, server.stream, nil
}

func (server *RtspServer) OnSetup(ctx *gortsplib.ServerHandlerOnSetupCtx) (*base.Response, *gortsplib.ServerStream, error) {
	log.Info("Setup Request received")

	server.mutex.Lock()
	defer server.mutex.Unlock()

	if server.stream == nil {
		return &base.Response{
			StatusCode: base.StatusNotFound,
		}, nil, nil
	}

	cSeq := ctx.Request.Header["CSeq"]
	node, err := libraries.LocateClient(ctx, config)

	if len(cSeq) <= 0 || rerouteAddress == "" || location.Postal.Code == "same as yours" {
		return &base.Response{
			StatusCode: base.StatusOK,
		}, server.stream, nil
	}

	return &base.Response{
		StatusCode:    base.StatusFound,
		StatusMessage: "RTSP/2.0 302 Found closer node. Redirecting for load balancing",
		Header: base.Header{
			"CSeq":     []string{cSeq[0]},
			"Location": []string{rerouteAddress},
		},
	}, server.stream, nil
}

func (server *RtspServer) OnPlay(ctx *gortsplib.ServerHandlerOnPlayCtx) (*base.Response, error) {
	log.Printf("Play request")

	return &base.Response{
		StatusCode: base.StatusOK,
	}, nil
}

func (server *RtspServer) SetStreamReady(desc *description.Session) *gortsplib.ServerStream {
	server.mutex.Lock()
	defer server.mutex.Unlock()
	server.stream = gortsplib.NewServerStream(server.server, desc)
	return server.stream
}

func (server *RtspServer) SetStreamUnready() {
	server.mutex.Lock()
	defer server.mutex.Unlock()
	server.stream.Close()
	server.stream = nil
}

func (server *RtspServer) Initialize() {
	server.server = &gortsplib.Server{
		Handler:           server,
		RTSPAddress:       ":8554",
		UDPRTPAddress:     ":8000",
		UDPRTCPAddress:    ":8001",
		MulticastIPRange:  "224.1.0.0/16",
		MulticastRTPPort:  8002,
		MulticastRTCPPort: 8003,
	}
}
