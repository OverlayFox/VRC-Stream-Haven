package multiplexer

import (
	"bytes"
	"io"
	"sync"
)

type syncBuffer struct {
	buf  *bytes.Buffer
	mu   sync.Mutex
	cond *sync.Cond
	err  error
}

func NewSyncBuffer(amountOfTSPackets int) *syncBuffer {
	rb := &syncBuffer{
		buf: bytes.NewBuffer(make([]byte, 0, amountOfTSPackets*MpegTsPktSize)),
	}
	rb.cond = sync.NewCond(&rb.mu)
	return rb
}

func (rb *syncBuffer) read(p []byte) (int, error) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	for {
		if rb.err != nil {
			return 0, rb.err
		}

		if rb.buf.Len() > 0 {
			return rb.buf.Read(p)
		}

		rb.cond.Wait()
	}
}

func (rb *syncBuffer) write(data []byte) error {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	defer rb.cond.Broadcast()

	if rb.err != nil {
		return rb.err
	}

	_, err := rb.buf.Write(data)
	return err
}

func (rb *syncBuffer) setError(err error) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	defer rb.cond.Broadcast()

	if rb.err == nil {
		rb.err = err
	}
}

func (rb *syncBuffer) Close() {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.err != nil {
		return
	}

	rb.err = io.EOF
	rb.cond.Broadcast()
}
