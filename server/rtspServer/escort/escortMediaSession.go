package flagship

import (
	"github.com/OverlayFox/VRC-Stream-Haven/logger"
	"github.com/OverlayFox/VRC-Stream-Haven/rtspServer"
	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"log"
	"reflect"
	"unsafe"
)

type EscortHandler struct {
	rtspServer.Handler
}

// GetReaders gets the readers map from a Stream instance using reflection.
func (eh *EscortHandler) GetReaders() (int, error) {
	streamReflect := reflect.ValueOf(eh.Stream).Elem()
	readersField := streamReflect.FieldByName("readers")
	readersField = reflect.NewAt(readersField.Type(), unsafe.Pointer(readersField.UnsafeAddr())).Elem()

	logger.HavenLogger.Info().Msgf("Readers: %v", readersField)

	return 1, nil
}

func (eh *EscortHandler) OnDescribe(ctx *gortsplib.ServerHandlerOnDescribeCtx) (*base.Response, *gortsplib.ServerStream, error) {
	log.Println("Describe Request")

	eh.Mutex.Lock()
	defer eh.Mutex.Unlock()

	if eh.Stream == nil {
		return &base.Response{
			StatusCode: base.StatusBadRequest,
		}, nil, nil
	}

	return &base.Response{
		StatusCode: base.StatusOK,
	}, eh.Stream, nil

}
