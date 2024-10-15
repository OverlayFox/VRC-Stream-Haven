package types

import (
	"fmt"
	"github.com/OverlayFox/VRC-Stream-Haven/geoLocator"
	"github.com/OverlayFox/VRC-Stream-Haven/harbor"
	"github.com/OverlayFox/VRC-Stream-Haven/logger"
	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"github.com/bluenviron/gortsplib/v4/pkg/description"
	"github.com/bluenviron/gortsplib/v4/pkg/format"
	"github.com/pion/rtp"
	"log"
	"net"
	"reflect"
	"strconv"
	"sync"
)

type RtspHandler struct {
	Server    *gortsplib.Server
	Stream    *gortsplib.ServerStream
	Publisher *gortsplib.ServerSession
	Mutex     sync.Mutex
}

// GetReaders gets the readers map from a Stream instance using reflection.
func (rh *RtspHandler) GetReaders() (int, error) {
	val := reflect.ValueOf(rh.Stream).Elem()

	readersField := val.FieldByName("readers")
	if !readersField.IsValid() {
		return 0, fmt.Errorf("field 'readers' not found")
	}

	readers, ok := readersField.Interface().(map[*gortsplib.ServerSession]struct{})
	if !ok {
		return 0, fmt.Errorf("could not convert 'readers' field to the expected type")
	}

	return len(readers), nil
}

func (rh *RtspHandler) OnConnectionOpen(ctx *gortsplib.ServerHandlerOnConnOpenCtx) {
	log.Println("Connection Opened")
}

func (rh *RtspHandler) OnConnectionClose(ctx *gortsplib.ServerHandlerOnConnCloseCtx) {
	log.Println("Connection Closed")
}

func (rh *RtspHandler) OnSessionOpen(ctx *gortsplib.ServerHandlerOnSessionOpenCtx) {
	log.Println("Session opened")
}

func (rh *RtspHandler) OnSessionClose(ctx *gortsplib.ServerHandlerOnSessionCloseCtx) {
	log.Println("Session closed")

	rh.Mutex.Lock()
	defer rh.Mutex.Unlock()

	if rh.Stream != nil && ctx.Session == rh.Publisher {
		rh.Stream.Close()
		rh.Stream = nil
	}
}

func (rh *RtspHandler) OnDescribe(ctx *gortsplib.ServerHandlerOnDescribeCtx) (*base.Response, *gortsplib.ServerStream, error) {
	log.Println("Describe Request")

	rh.Mutex.Lock()
	defer rh.Mutex.Unlock()

	if rh.Stream == nil {
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
			}, rh.Stream, nil
		}

		closestEscorts := harbor.Haven.GetClosestEscort(city)
		if closestEscorts[0].IpAddress.Equal(harbor.Haven.Flagship.IpAddress) {
			return &base.Response{
				StatusCode: base.StatusOK,
			}, rh.Stream, nil
		}

		for _, escort := range closestEscorts {
			if !escort.CheckAvailability() {
				logger.HavenLogger.Warn().Msgf("Escort %s is not available. Removing from Haven", escort.IpAddress.String())
				err := harbor.Haven.RemoveEscort(escort.IpAddress)
				if err != nil {
					logger.HavenLogger.Warn().Err(err).Msgf("Failed to remove escort %s from Haven", escort.IpAddress.String())
				}
				continue
			}

			if !escort.MaxReadersReached() {
				logger.HavenLogger.Info().Msgf("Escort %s has reached the maximum number of readers", escort.IpAddress.String())
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
	}, rh.Stream, nil

}

func (rh *RtspHandler) OnAnnounce(ctx *gortsplib.ServerHandlerOnAnnounceCtx) (*base.Response, error) {
	log.Println("Announce Request")

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

func (rh *RtspHandler) OnSetup(ctx *gortsplib.ServerHandlerOnSetupCtx) (*base.Response, *gortsplib.ServerStream, error) {
	log.Println("Setup Request")

	if rh.Stream == nil {
		return &base.Response{
			StatusCode: base.StatusNotFound,
		}, nil, nil
	}

	return &base.Response{
		StatusCode: base.StatusOK,
	}, rh.Stream, nil
}

func (rh *RtspHandler) OnRecord(ctx *gortsplib.ServerHandlerOnRecordCtx) (*base.Response, error) {
	log.Println("Record Request")

	ctx.Session.OnPacketRTPAny(func(media *description.Media, format format.Format, packet *rtp.Packet) {
		rh.Stream.WritePacketRTP(media, packet)
	})

	return &base.Response{
		StatusCode: base.StatusOK,
	}, nil
}

func (rh *RtspHandler) OnPlay(ctx *gortsplib.ServerHandlerOnPlayCtx) (*base.Response, error) {
	log.Println("Play Request")

	if rh.Stream != nil {
		return &base.Response{
			StatusCode: base.StatusOK,
		}, nil
	}

	return &base.Response{
		StatusCode: base.StatusNotFound,
	}, nil
}
