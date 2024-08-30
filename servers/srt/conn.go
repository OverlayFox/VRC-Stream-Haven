package srt

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/OverlayFox/VRC-Stream-Haven/logger"
	"net"
	"sync"
	"time"

	srt "github.com/datarhei/gosrt"
	"github.com/google/uuid"
)

func srtCheckPassphrase(connReq srt.ConnRequest, passphrase string) error {
	if passphrase == "" {
		return nil
	}

	if !connReq.IsEncrypted() {
		return fmt.Errorf("connection is encrypted, but not passphrase is defined in configuration")
	}

	err := connReq.SetPassphrase(passphrase)
	if err != nil {
		return fmt.Errorf("invalid passphrase")
	}

	return nil
}

type connState int

const (
	connStateRead connState = iota + 1
	connStatePublish
)

type conn struct {
	parentCtx         context.Context
	readTimeout       time.Duration
	writeTimeout      time.Duration
	writeQueueSize    int
	udpMaxPayloadSize int
	connReq           srt.ConnRequest
	wg                *sync.WaitGroup
	parent            *Server

	ctx       context.Context
	ctxCancel func()
	created   time.Time
	uuid      uuid.UUID
	mutex     sync.RWMutex
	state     connState
	pathName  string
	query     string
	sconn     srt.Conn

	chNew     chan srtNewConnReq
	chSetConn chan srt.Conn
}

func (c *conn) initialize() {
	c.ctx, c.ctxCancel = context.WithCancel(c.parentCtx)

	c.created = time.Now()
	c.uuid = uuid.New()
	c.chNew = make(chan srtNewConnReq)
	c.chSetConn = make(chan srt.Conn)

	logger.Log.Info().Msg("opened")

	c.wg.Add(1)
	go c.run()
}

func (c *conn) Close() {
	c.ctxCancel()
}

func (c *conn) ip() net.IP {
	return c.connReq.RemoteAddr().(*net.UDPAddr).IP
}

func (c *conn) run() { //nolint:dupl
	defer c.wg.Done()

	err := c.runInner()

	c.ctxCancel()

	c.parent.closeConn(c)

	logger.Log.Info().Err(err).Msgf("closed")
}

func (c *conn) runInner() error {
	var req srtNewConnReq
	select {
	case req = <-c.chNew:
	case <-c.ctx.Done():
		return errors.New("terminated")
	}

	answerSent, err := c.runInner2(req)

	if !answerSent {
		req.res <- nil
	}

	return err
}

func (c *conn) runInner2(req srtNewConnReq) (bool, error) {
	var streamID streamID
	err := streamID.unmarshal(req.connReq.StreamId())
	if err != nil {
		return false, fmt.Errorf("invalid stream ID '%s': %w", req.connReq.StreamId(), err)
	}

	if streamID.mode == streamIDModePublish {
		return c.runPublish(req, &streamID)
	}
	return c.runRead(req, &streamID)
}

func (c *conn) runPublish(req srtNewConnReq, streamID *streamID) (bool, error) {
	err := srtCheckPassphrase(req.connReq, "helloworldhowareyou")
	if err != nil {
		return false, err
	}

	sconn, err := c.exchangeRequestWithConn(req)
	if err != nil {
		return true, err
	}

	c.mutex.Lock()
	c.state = connStatePublish
	c.pathName = streamID.path
	c.query = streamID.query
	c.sconn = sconn
	c.mutex.Unlock()

	readerErr := make(chan error)
	// @Todo: Implement a way to read from the stream and push it to somewhere else
	//go func() {
	//	readerErr <- c.runPublishReader(sconn, path)
	//}()

	select {
	case err := <-readerErr:
		sconn.Close()
		return true, err

	case <-c.ctx.Done():
		sconn.Close()
		<-readerErr
		return true, errors.New("terminated")
	}
}

// @Todo: Implement a way to read from the stream and push it to somewhere else
//func (c *conn) runPublishReader(sconn srt.Conn) error {
//	sconn.SetReadDeadline(time.Now().Add(time.Duration(c.readTimeout)))
//	r, err := mcmpegts.NewReader(mcmpegts.NewBufferedReader(sconn))
//	if err != nil {
//		return err
//	}
//
//	decodeErrLogger := logger.Log
//
//	r.OnDecodeError(func(err error) {
//		decodeErrLogger.Warn().Err(err)
//	})
//
//	var stream *stream.Stream
//
//	medias, err := mpegts.ToStream(r, &stream)
//	if err != nil {
//		return err
//	}
//
//	stream, err = path.StartPublisher(defs.PathStartPublisherReq{
//		Author:             c,
//		Desc:               &description.Session{Medias: medias},
//		GenerateRTPPackets: true,
//	})
//	if err != nil {
//		return err
//	}
//
//	for {
//		err := r.Read()
//		if err != nil {
//			return err
//		}
//	}
//}

func (c *conn) runRead(req srtNewConnReq, streamID *streamID) (bool, error) {
	err := srtCheckPassphrase(req.connReq, "helloworldhowareyou")
	if err != nil {
		return false, err
	}

	sconn, err := c.exchangeRequestWithConn(req)
	if err != nil {
		return true, err
	}
	defer sconn.Close()

	c.mutex.Lock()
	c.state = connStateRead
	c.pathName = streamID.path
	c.query = streamID.query
	c.sconn = sconn
	c.mutex.Unlock()

	writer := asyncwriter.New(c.writeQueueSize, c)

	defer stream.RemoveReader(writer)

	bw := bufio.NewWriterSize(sconn, srtMaxPayloadSize(c.udpMaxPayloadSize))

	err = mpegts.FromStream(stream, writer, bw, sconn, time.Duration(c.writeTimeout))
	if err != nil {
		return true, err
	}

	logger.Log.Info().Msgf("is reading from path")

	// disable read deadline
	sconn.SetReadDeadline(time.Time{})

	select {
	case <-c.ctx.Done():
		return true, fmt.Errorf("terminated")
	}
}

func (c *conn) exchangeRequestWithConn(req srtNewConnReq) (srt.Conn, error) {
	req.res <- c

	select {
	case sconn := <-c.chSetConn:
		return sconn, nil

	case <-c.ctx.Done():
		return nil, errors.New("terminated")
	}
}

// new is called by srtListener through srtServer.
func (c *conn) new(req srtNewConnReq) *conn {
	select {
	case c.chNew <- req:
		return <-req.res

	case <-c.ctx.Done():
		return nil
	}
}

// setConn is called by srtListener .
func (c *conn) setConn(sconn srt.Conn) {
	select {
	case c.chSetConn <- sconn:
	case <-c.ctx.Done():
	}
}
