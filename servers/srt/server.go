package srt

import (
	"errors"
	"github.com/OverlayFox/VRC-Stream-Haven/logger"
	srt "github.com/datarhei/gosrt"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
)

type Server struct {
	addr       string
	app        string
	passphrase string
	logtopics  string
	profile    string

	server *srt.Server

	channels map[string]srt.PubSub
	lock     sync.RWMutex
}

// listenAndServe starts the SRT server.
// It blocks until an error happens.
// If the error is ErrServerClosed the server has shutdown normally.
func (s *Server) listenAndServe() error {
	if len(s.app) == 0 {
		s.app = "/"
	}

	return s.server.ListenAndServe()
}

func (s *Server) log(who, action, path, message string, client net.Addr) {
	logger.Log.Error().Msgf("%s %s %s %s %s", who, action, path, message, client)
}

func (s *Server) handleConnect(req srt.ConnRequest) {
	client := req.RemoteAddr()

	channel := ""

	if req.Version() != 5 {
		s.log("Connect", "Forbidden", req.StreamId(), "Unsupported version", client)
		req.Reject(srt.REJ_VERSION)
		return
	}

	if !strings.HasPrefix(req.StreamId(), "publish:") {
		accept, err := req.Accept()
		if err != nil {
			s.log("Connect", "Error", req.StreamId(), "Could not accept incoming connection", client)
			return
		}

		accept.
	}
}

func Initialize() {
	s := Server{
		channels: make(map[string]srt.PubSub),
	}

	s.logtopics = "connection:close,connection:error,connection:new,listen"

	config := srt.DefaultConfig()
	config.Logger = srt.NewLogger(strings.Split(s.logtopics, ","))
	config.KMPreAnnounce = 200
	config.KMRefreshRate = 10000

	s.server = &srt.Server{
		Addr:            "0.0.0.0:6001",
		Config:          &config,
		HandleConnect:   nil,
		HandlePublish:   nil,
		HandleSubscribe: nil,
	}

	go func() {
		if config.Logger == nil {
			return
		}

		for m := range config.Logger.Listen() {
			logger.Log.Info().Msgf("%#08x %s (in %s:%d)\n%s \n", m.SocketId, m.Topic, m.File, m.Line, m.Message)
		}
	}()

	go func() {
		err := s.listenAndServe()
		if err != nil && !errors.Is(err, srt.ErrServerClosed) {
			logger.Log.Fatal().Err(err).Msg("Failed to start SRT server or a unexpected error occurred")
		}
	}()

	logger.Log.Info().Msgf("SRT Server is listening on %s", s.server.Addr)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	s.server.Shutdown()

	if config.Logger != nil {
		config.Logger.Close()
	}
}
