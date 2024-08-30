package srt

import (
	"errors"
	"flag"
	"fmt"
	"github.com/OverlayFox/VRC-Stream-Haven/logger"
	"github.com/OverlayFox/VRC-Stream-Haven/servers/srt/types"
	gosrt "github.com/datarhei/gosrt"
	"github.com/pkg/profile"
	"os"
	"os/signal"
	"strconv"
	"strings"
)

func srtMaxPayloadSize(u int) int {
	return ((u - 16) / 188) * 188 // 16 = SRT header, 188 = MPEG-TS packet
}

func StartUpIngestSRT(port uint16, passphrase string) *types.MediaSession {
	s := &types.MediaSession{
		Channels: make(map[string]gosrt.PubSub),
	}

	s.Addr = "192.168.0.42:" + strconv.Itoa(int(port))
	s.App = "/haven"
	s.Passphrase = passphrase
	s.Profile = "cpu"

	flag.Parse()

	if len(s.Addr) == 0 {
		fmt.Fprintf(os.Stderr, "Provide a listen address with -addr\n")
		os.Exit(1)
	}

	var p func(*profile.Profile)
	switch s.Profile {
	case "cpu":
		p = profile.CPUProfile
	case "mem":
		p = profile.MemProfile
	case "allocs":
		p = profile.MemProfileAllocs
	case "heap":
		p = profile.MemProfileHeap
	case "rate":
		p = profile.MemProfileRate(2048)
	case "mutex":
		p = profile.MutexProfile
	case "block":
		p = profile.BlockProfile
	case "thread":
		p = profile.ThreadcreationProfile
	case "trace":
		p = profile.TraceProfile
	default:
	}

	if p != nil {
		defer profile.Start(profile.ProfilePath("."), profile.NoShutdownHook, p).Stop()
	}

	config := gosrt.DefaultConfig()
	config.PayloadSize = uint32(srtMaxPayloadSize(1472))
	config.EnforcedEncryption = true

	if len(s.LogTopics) != 0 {
		config.Logger = gosrt.NewLogger(strings.Split(s.LogTopics, ","))
	}

	s.Server = &gosrt.Server{
		Addr:            s.Addr,
		HandleConnect:   s.HandleConnect,
		HandlePublish:   s.HandlePublish,
		HandleSubscribe: s.HandleSubscribe,
		Config:          &config,
	}

	logger.Log.Info().Msgf("Listening on %s\n", s.Addr)

	go func() {
		if config.Logger == nil {
			return
		}

		for m := range config.Logger.Listen() {
			logger.Log.Info().Msgf("%#08x %s (in %s:%d)\n%s \n", m.SocketId, m.Topic, m.File, m.Line, m.Message)
		}
	}()

	go func() {
		if err := s.ListenAndServe(); err != nil && !errors.Is(err, gosrt.ErrServerClosed) {
			logger.Log.Info().Msgf("SRT Server: %s\n", err)
			os.Exit(2)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	s.Shutdown()

	if config.Logger != nil {
		config.Logger.Close()
	}

	return s
}
