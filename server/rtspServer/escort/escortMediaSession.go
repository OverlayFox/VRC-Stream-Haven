package flagship

import (
	"fmt"
	"github.com/OverlayFox/VRC-Stream-Haven/rtspServer"
	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"log"
	"reflect"
)

type EscortHandler struct {
	rtspServer.Handler
}

// GetReaders gets the readers map from a Stream instance using reflection.
func (eh *EscortHandler) GetReaders() (int, error) {
	val := reflect.ValueOf(eh.Stream).Elem()

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
