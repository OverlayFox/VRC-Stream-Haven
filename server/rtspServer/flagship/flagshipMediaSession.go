package flagship

import (
	"github.com/OverlayFox/VRC-Stream-Haven/apiService/flagship"
	"github.com/OverlayFox/VRC-Stream-Haven/geoLocator"
	"github.com/OverlayFox/VRC-Stream-Haven/harbor"
	"github.com/OverlayFox/VRC-Stream-Haven/logger"
	"github.com/OverlayFox/VRC-Stream-Haven/rtspServer"
	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"log"
	"net"
	"strconv"
)

type FlagshipHandler struct {
	rtspServer.Handler
}

func (fh *FlagshipHandler) OnDescribe(ctx *gortsplib.ServerHandlerOnDescribeCtx) (*base.Response, *gortsplib.ServerStream, error) {
	log.Println("Describe Request")

	fh.Mutex.Lock()
	defer fh.Mutex.Unlock()

	if fh.Stream == nil {
		return &base.Response{
			StatusCode: base.StatusBadRequest,
		}, nil, nil
	}

	if harbor.Haven.IsServer {
		clientIp := ctx.Conn.NetConn().RemoteAddr().(*net.TCPAddr).IP

		city, err := geoLocator.LocateIp(clientIp.String())
		if err != nil {
			logger.HavenLogger.Warn().Err(err).Msg("Failed to locate IP of the client. Redirecting to Flagship")
			return &base.Response{
				StatusCode: base.StatusOK,
			}, fh.Stream, nil
		}

		closestEscorts := harbor.Haven.GetClosestEscort(city)
		if closestEscorts[0].IpAddress.Equal(harbor.Haven.Flagship.IpAddress) {
			return &base.Response{
				StatusCode: base.StatusOK,
			}, fh.Stream, nil
		}

		for _, escort := range closestEscorts {
			if !flagship.IsApiOnline(escort) {
				logger.HavenLogger.Warn().Msgf("Escort %s is not reachable. Removing from Haven", escort.IpAddress.String())
				err := harbor.Haven.RemoveEscort(escort.IpAddress)
				if err != nil {
					logger.HavenLogger.Warn().Err(err).Msgf("Failed to remove escort %s from Haven", escort.IpAddress.String())
				}
				continue
			}

			readers, err := flagship.GetEscortReaders(escort)
			if err != nil {
				logger.HavenLogger.Error().Err(err).Msgf("Failed to get readers for escort %s", escort.IpAddress.String())
				return &base.Response{
					StatusCode: base.StatusOK,
				}, fh.Stream, nil
			}

			if readers.CurrentViewers >= readers.MaxAllowedViewers {
				logger.HavenLogger.Info().Msgf(
					"Escort %s has reached the maximum number of readers. Current viewers: %d. Maxiumum allowed readers: %d",
					escort.IpAddress.String(), readers.CurrentViewers, readers.MaxAllowedViewers)
				continue
			}
			logger.HavenLogger.Info().Msgf("Redirecting to %s", escort.IpAddress.String())

			return &base.Response{
				StatusCode: base.StatusMovedPermanently,
				Header: base.Header{
					"Location": base.HeaderValue{"rtsp://" + escort.IpAddress.String() + ":" + strconv.FormatUint(uint64(escort.RtspEgressPort), 10)},
				},
			}, nil, nil
		}
	}

	return &base.Response{
		StatusCode: base.StatusOK,
	}, fh.Stream, nil

}
