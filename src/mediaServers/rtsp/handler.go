package rtsp

import (
	"fmt"
	"strings"
	"sync"

	"github.com/OverlayFox/VRC-Stream-Haven/src/types"
	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"github.com/bluenviron/gortsplib/v4/pkg/description"
	"github.com/bluenviron/gortsplib/v4/pkg/format"
	"github.com/pion/rtp"
	"github.com/rs/zerolog"
)

type Handler struct {
	server    *gortsplib.Server
	stream    *gortsplib.ServerStream
	publisher *gortsplib.ServerSession

	passphrase string

	governor types.Governor

	mtx    sync.RWMutex
	logger zerolog.Logger
}

func NewHandler(server *gortsplib.Server, governor types.Governor, passphrase string, logger zerolog.Logger) Handler {
	return Handler{
		server:     server,
		governor:   governor,
		passphrase: passphrase,
		logger:     logger,
	}
}

func (h *Handler) OnConnOpen(ctx *gortsplib.ServerHandlerOnConnOpenCtx) {
	h.logger.Info().Msg("Connection Opened")
}

func (h *Handler) OnConnClose(ctx *gortsplib.ServerHandlerOnConnCloseCtx) {
	h.logger.Info().Msg("Connection Closed")
}

func (h *Handler) OnSessionOpen(ctx *gortsplib.ServerHandlerOnSessionOpenCtx) {
	h.logger.Info().Msg("Session opened")
}

func (h *Handler) OnSessionClose(ctx *gortsplib.ServerHandlerOnSessionCloseCtx) {
	h.logger.Info().Msg("Session closed")

	h.mtx.Lock()
	defer h.mtx.Unlock()

	if h.stream != nil && ctx.Session == h.publisher {
		h.stream.Close()
		h.stream = nil
	}
}

func (h *Handler) OnDescribe(ctx *gortsplib.ServerHandlerOnDescribeCtx) (*base.Response, *gortsplib.ServerStream, error) {
	h.logger.Info().Msg("Describe Request")

	h.mtx.Lock()
	defer h.mtx.Unlock()

	if ctx.Path != fmt.Sprintf("/%s", h.passphrase) {
		return &base.Response{
			StatusCode: base.StatusConnectionCredentialsNotAccepted,
		}, nil, nil
	}

	return &base.Response{
		StatusCode: base.StatusOK,
	}, h.stream, nil
}

func (h *Handler) OnAnnounce(ctx *gortsplib.ServerHandlerOnAnnounceCtx) (*base.Response, error) {
	h.logger.Info().Msg("Announce Request")

	h.mtx.Lock()
	defer h.mtx.Unlock()

	paths := strings.Split(ctx.Path, "/")
	if len(paths) < 3 {
		return &base.Response{
			StatusCode: base.StatusConnectionCredentialsNotAccepted,
		}, nil
	}
	// streamId := paths[1]
	passphrase := paths[2]

	if passphrase != h.passphrase {
		return &base.Response{
			StatusCode: base.StatusConnectionCredentialsNotAccepted,
		}, nil
	}

	if h.stream != nil {
		return &base.Response{
			StatusCode: base.StatusBadRequest,
		}, nil
	}
	stream := gortsplib.NewServerStream(h.server, ctx.Description)
	h.stream = stream
	h.publisher = ctx.Session

	return &base.Response{
		StatusCode: base.StatusOK,
	}, nil
}

func (h *Handler) OnSetup(ctx *gortsplib.ServerHandlerOnSetupCtx) (*base.Response, *gortsplib.ServerStream, error) {
	h.logger.Info().Msg("Setup Request")

	if h.stream == nil {
		return &base.Response{
			StatusCode: base.StatusNotFound,
		}, nil, nil
	}

	return &base.Response{
		StatusCode: base.StatusOK,
	}, h.stream, nil
}

func (h *Handler) OnPlay(ctx *gortsplib.ServerHandlerOnPlayCtx) (*base.Response, error) {
	h.logger.Info().Msg("Play Request")

	if h.stream != nil {
		return &base.Response{
			StatusCode: base.StatusOK,
		}, nil
	}

	return &base.Response{
		StatusCode: base.StatusNotFound,
	}, nil
}

func (h *Handler) OnRecord(ctx *gortsplib.ServerHandlerOnRecordCtx) (*base.Response, error) {
	h.logger.Info().Msg("Record Request")

	ctx.Session.OnPacketRTPAny(func(media *description.Media, format format.Format, packet *rtp.Packet) {
		h.stream.WritePacketRTP(media, packet)
	})

	return &base.Response{
		StatusCode: base.StatusOK,
	}, nil
}
