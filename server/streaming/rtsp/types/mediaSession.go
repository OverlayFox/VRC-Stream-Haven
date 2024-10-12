package rtspServer

import (
	"github.com/OverlayFox/VRC-Stream-Haven/geoLocator"
	"github.com/OverlayFox/VRC-Stream-Haven/harbor"
	"github.com/OverlayFox/VRC-Stream-Haven/logger"
	"log"
	"net"
	"strconv"
	"sync"

	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"github.com/bluenviron/gortsplib/v4/pkg/description"
	"github.com/bluenviron/gortsplib/v4/pkg/format"
	"github.com/pion/rtp"
)

type RtspMediaSession struct {
	Server    *gortsplib.Server
	Stream    *gortsplib.ServerStream
	Publisher *gortsplib.ServerSession
	Mutex     sync.Mutex
}

func (sh *RtspMediaSession) OnConnectionOpen(ctx *gortsplib.ServerHandlerOnConnOpenCtx) {
	log.Println("Connection Opened")
}

func (sh *RtspMediaSession) OnConnectionClose(ctx *gortsplib.ServerHandlerOnConnCloseCtx) {
	log.Println("Connection Closed")
}

func (sh *RtspMediaSession) OnSessionOpen(ctx *gortsplib.ServerHandlerOnSessionOpenCtx) {
	log.Println("Session opened")
}

func (sh *RtspMediaSession) OnSessionClose(ctx *gortsplib.ServerHandlerOnSessionCloseCtx) {
	log.Println("Session closed")

	sh.Mutex.Lock()
	defer sh.Mutex.Unlock()

	if sh.Stream != nil && ctx.Session == sh.Publisher {
		sh.Stream.Close()
		sh.Stream = nil
	}
}

func (sh *RtspMediaSession) OnDescribe(ctx *gortsplib.ServerHandlerOnDescribeCtx) (*base.Response, *gortsplib.ServerStream, error) {
	log.Println("Describe Request")

	sh.Mutex.Lock()
	defer sh.Mutex.Unlock()

	if sh.Stream == nil {
		return &base.Response{
			StatusCode: base.StatusBadRequest,
		}, nil, nil
	}

	if harbor.Haven.IsServer {
		clientIp := ctx.Conn.NetConn().RemoteAddr().(*net.TCPAddr).IP

		city, err := geoLocator.LocateIp(clientIp.String())
		if err != nil {
			logger.HavenLogger.Warn().Err(err).Msg("Failed to locate IP")
			return &base.Response{
				StatusCode: base.StatusOK,
			}, sh.Stream, nil
		}

		closestEscort := harbor.Haven.GetClosestEscort(city)
		if closestEscort == harbor.Haven.Flagship.Ship {
			return &base.Response{
				StatusCode: base.StatusOK,
			}, sh.Stream, nil
		}

		logger.HavenLogger.Info().Msgf("Redirecting to %s", closestEscort.Username)
		return &base.Response{
			StatusCode: base.StatusMovedPermanently,
			Header: base.Header{
				"Location": base.HeaderValue{"rtsp://" + closestEscort.IpAddress.String() + ":" + strconv.FormatUint(uint64(closestEscort.RtspEgressPort), 10)},
			},
		}, nil, nil
	}

	return &base.Response{
		StatusCode: base.StatusOK,
	}, sh.Stream, nil

}

func (sh *RtspMediaSession) OnAnnounce(ctx *gortsplib.ServerHandlerOnAnnounceCtx) (*base.Response, error) {
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

func (sh *RtspMediaSession) OnSetup(ctx *gortsplib.ServerHandlerOnSetupCtx) (*base.Response, *gortsplib.ServerStream, error) {
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

func (sh *RtspMediaSession) OnRecord(ctx *gortsplib.ServerHandlerOnRecordCtx) (*base.Response, error) {
	log.Println("Record Request")

	ctx.Session.OnPacketRTPAny(func(media *description.Media, format format.Format, packet *rtp.Packet) {
		sh.Stream.WritePacketRTP(media, packet)
	})

	return &base.Response{
		StatusCode: base.StatusOK,
	}, nil
}

func (sh *RtspMediaSession) OnPlay(ctx *gortsplib.ServerHandlerOnPlayCtx) (*base.Response, error) {
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
