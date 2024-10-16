package rtspServer

import (
	"github.com/OverlayFox/VRC-Stream-Haven/logger"
	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"github.com/bluenviron/gortsplib/v4/pkg/description"
	"github.com/bluenviron/gortsplib/v4/pkg/format"
	"github.com/pion/rtp"
	"sync"
)

type Handler struct {
	Server    *gortsplib.Server
	Stream    *gortsplib.ServerStream
	Publisher *gortsplib.ServerSession
	Mutex     sync.Mutex
}

func (rh *Handler) OnConnectionOpen(ctx *gortsplib.ServerHandlerOnConnOpenCtx) {
	logger.HavenLogger.Info().Msg("Connection Opened")
}

func (rh *Handler) OnConnectionClose(ctx *gortsplib.ServerHandlerOnConnCloseCtx) {
	logger.HavenLogger.Info().Msg("Connection Closed")
}

func (rh *Handler) OnSessionOpen(ctx *gortsplib.ServerHandlerOnSessionOpenCtx) {
	logger.HavenLogger.Info().Msg("Session opened")
}

func (rh *Handler) OnAnnounce(ctx *gortsplib.ServerHandlerOnAnnounceCtx) (*base.Response, error) {
	logger.HavenLogger.Info().Msg("Announce Request")

	rh.Mutex.Lock()
	defer rh.Mutex.Unlock()

	if rh.Stream != nil {
		rh.Stream.Close()
		rh.Publisher.Close()
	}

	rh.Stream = gortsplib.NewServerStream(rh.Server, ctx.Description)
	rh.Publisher = ctx.Session

	return &base.Response{
		StatusCode: base.StatusOK,
	}, nil
}

func (rh *Handler) OnSetup(ctx *gortsplib.ServerHandlerOnSetupCtx) (*base.Response, *gortsplib.ServerStream, error) {
	logger.HavenLogger.Info().Msg("Setup Request")

	if rh.Stream == nil {
		return &base.Response{
			StatusCode: base.StatusNotFound,
		}, nil, nil
	}

	return &base.Response{
		StatusCode: base.StatusOK,
	}, rh.Stream, nil
}

func (rh *Handler) OnRecord(ctx *gortsplib.ServerHandlerOnRecordCtx) (*base.Response, error) {
	logger.HavenLogger.Info().Msg("Record Request")

	ctx.Session.OnPacketRTPAny(func(media *description.Media, format format.Format, packet *rtp.Packet) {
		rh.Stream.WritePacketRTP(media, packet)
	})

	return &base.Response{
		StatusCode: base.StatusOK,
	}, nil
}

func (rh *Handler) OnPlay(ctx *gortsplib.ServerHandlerOnPlayCtx) (*base.Response, error) {
	logger.HavenLogger.Info().Msg("Play Request")

	if rh.Stream != nil {
		return &base.Response{
			StatusCode: base.StatusOK,
		}, nil
	}

	return &base.Response{
		StatusCode: base.StatusNotFound,
	}, nil
}

func (rh *Handler) OnSessionClose(ctx *gortsplib.ServerHandlerOnSessionCloseCtx) {
	logger.HavenLogger.Info().Msg("Session closed")

	rh.Mutex.Lock()
	defer rh.Mutex.Unlock()

	if rh.Stream != nil && ctx.Session == rh.Publisher {
		rh.Stream.Close()
		rh.Stream = nil
	}
}
