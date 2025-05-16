package rtsp

import (
	"strings"

	"github.com/OverlayFox/VRC-Stream-Haven/src/types"
	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"github.com/rs/zerolog"
)

type FlagshipHandler struct {
	Handler
}

func NewFlagshipHandler(server *gortsplib.Server, governor types.Governor, passphrase string, logger zerolog.Logger) FlagshipHandler {
	return FlagshipHandler{
		Handler: Handler{
			server:     server,
			governor:   governor,
			passphrase: passphrase,
			logger:     logger,
		},
	}
}

// OnDescribes overwrite the original handler OnDescribe method to handle redirections.
func (h *FlagshipHandler) OnDescribe(ctx *gortsplib.ServerHandlerOnDescribeCtx) (*base.Response, *gortsplib.ServerStream, error) {
	h.logger.Info().Msg("Describe Request")

	h.mtx.Lock()
	defer h.mtx.Unlock()

	paths := strings.Split(ctx.Path, "/")
	if len(paths) < 3 {
		return &base.Response{
			StatusCode: base.StatusConnectionCredentialsNotAccepted,
		}, nil, nil
	}
	streamId := strings.TrimSpace(paths[1])
	passphrase := strings.TrimSpace(paths[2])

	clientIp := ctx.Conn.NetConn().RemoteAddr()
	h.logger.Debug().Msgf("Received read request from IP '%s' for haven '%s'", clientIp.String(), streamId)

	haven, err := h.governor.GetHaven(streamId)
	if err != nil {
		h.logger.Warn().Msgf("Requested haven '%s' does not exist: %v", streamId, err)
		return &base.Response{
			StatusCode: base.StatusNotFound,
		}, nil, nil
	}

	if haven.GetPassphrase() != passphrase {
		h.logger.Warn().Msgf("Passphrase mismatch for client '%s': expected '%s', got '%s'", clientIp.String(), haven.GetPassphrase(), passphrase)
		return &base.Response{
			StatusCode: base.StatusConnectionCredentialsNotAccepted,
		}, nil, nil
	}

	escort, err := haven.GetClosestEscort(clientIp)
	if err != nil {
		h.logger.Warn().Msgf("Failed to get escort for client '%s': %v. Letting client pass to flagship", clientIp.String(), err)
		return &base.Response{
			StatusCode: base.StatusOK,
		}, h.stream, nil
	}

	h.logger.Info().Msgf("Redirecting client '%s' to escort '%s'", clientIp.String(), escort.GetAddr().String())
	return &base.Response{
		StatusCode: base.StatusMovedPermanently,
		Header: base.Header{
			"Location": base.HeaderValue{"rtsp://" + escort.GetAddr().String() + ctx.Path},
		},
	}, nil, nil
}
