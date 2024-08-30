// Package srt contains a SRT server.
package srt

import (
	"context"
	"errors"
	"github.com/OverlayFox/VRC-Stream-Haven/logger"
	"github.com/rs/zerolog"
	"sync"
	"time"

	srt "github.com/datarhei/gosrt"
)

// ErrConnNotFound is returned when a connection is not found.
var ErrConnNotFound = errors.New("connection not found")

func srtMaxPayloadSize(u int) int {
	return ((u - 16) / 188) * 188 // 16 = SRT header, 188 = MPEG-TS packet
}

type srtNewConnReq struct {
	connReq srt.ConnRequest
	res     chan *conn
}

// Server is a SRT server.
type Server struct {
	Address           string
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	WriteQueueSize    int
	UDPMaxPayloadSize int
	Logger            *zerolog.Logger

	ctx       context.Context
	ctxCancel func()
	wg        sync.WaitGroup
	ln        srt.Listener
	conns     map[*conn]struct{}

	// in
	chNewConnRequest chan srtNewConnReq
	chAcceptErr      chan error
	chCloseConn      chan *conn
}

// Initialize initializes the server.
func (s *Server) Initialize() error {
	conf := srt.DefaultConfig()
	conf.ConnectionTimeout = s.ReadTimeout
	conf.PayloadSize = uint32(srtMaxPayloadSize(s.UDPMaxPayloadSize))

	var err error
	s.ln, err = srt.Listen("srt", s.Address, conf)
	if err != nil {
		return err
	}

	s.ctx, s.ctxCancel = context.WithCancel(context.Background())

	s.conns = make(map[*conn]struct{})
	s.chNewConnRequest = make(chan srtNewConnReq)
	s.chAcceptErr = make(chan error)
	s.chCloseConn = make(chan *conn)

	logger.Log.Info().Msgf("listener opened on %s (SRT)", s.Address)

	l := &listener{
		ln:     s.ln,
		wg:     &s.wg,
		parent: s,
	}
	l.initialize()

	s.wg.Add(1)
	go s.run()

	return nil
}

// Close closes the server.
func (s *Server) Close() {
	logger.Log.Info().Msgf("listener is closing")
	s.ctxCancel()
	s.wg.Wait()
}

func (s *Server) run() {
	defer s.wg.Done()

outer:
	for {
		select {
		case err := <-s.chAcceptErr:
			logger.Log.Error().Err(err)
			break outer

		case req := <-s.chNewConnRequest:
			c := &conn{
				parentCtx:         s.ctx,
				readTimeout:       s.ReadTimeout,
				writeTimeout:      s.WriteTimeout,
				writeQueueSize:    s.WriteQueueSize,
				udpMaxPayloadSize: s.UDPMaxPayloadSize,
				connReq:           req.connReq,
				wg:                &s.wg,
				parent:            s,
			}
			c.initialize()
			s.conns[c] = struct{}{}
			req.res <- c

		case c := <-s.chCloseConn:
			delete(s.conns, c)

		case <-s.ctx.Done():
			break outer
		}
	}

	s.ctxCancel()

	s.ln.Close()
}

// newConnRequest is called by srtListener.
func (s *Server) newConnRequest(connReq srt.ConnRequest) *conn {
	req := srtNewConnReq{
		connReq: connReq,
		res:     make(chan *conn),
	}

	select {
	case s.chNewConnRequest <- req:
		c := <-req.res

		return c.new(req)

	case <-s.ctx.Done():
		return nil
	}
}

// acceptError is called by srtListener.
func (s *Server) acceptError(err error) {
	select {
	case s.chAcceptErr <- err:
	case <-s.ctx.Done():
	}
}

// closeConn is called by conn.
func (s *Server) closeConn(c *conn) {
	select {
	case s.chCloseConn <- c:
	case <-s.ctx.Done():
	}
}
