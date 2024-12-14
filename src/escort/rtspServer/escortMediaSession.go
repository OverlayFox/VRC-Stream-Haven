package rtspServer

import (
	"fmt"
	"github.com/OverlayFox/VRC-Stream-Haven/shared/rtspServer"
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
	if eh.Stream == nil {
		return 0, fmt.Errorf("stream is nil")
	}

	streamReflect := reflect.ValueOf(eh.Stream)
	if !streamReflect.IsValid() || streamReflect.Kind() != reflect.Ptr || streamReflect.IsNil() {
		return 0, fmt.Errorf("invalid Stream value")
	}

	streamReflect = streamReflect.Elem()
	readersField := streamReflect.FieldByName("readers")
	if !readersField.IsValid() {
		return 0, fmt.Errorf("field 'readers' not found")
	}

	readersField = reflect.NewAt(readersField.Type(), unsafe.Pointer(readersField.UnsafeAddr())).Elem()
	if readersField.Kind() != reflect.Map {
		return 0, fmt.Errorf("field 'readers' is not a map")
	}

	return readersField.Len(), nil
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
