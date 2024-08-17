package srtServer

import (
	"errors"
	"flag"
	"fmt"
	utils "github.com/OverlayFox/VRC-Stream-Haven/servers"
	"github.com/OverlayFox/VRC-Stream-Haven/servers/srtServer/types"
	srt "github.com/datarhei/gosrt"
	"github.com/pkg/profile"
	"os"
	"os/signal"
	"strconv"
	"strings"
)

func SetupBackendSrt(passphrase string) *types.MediaSession {
	s := &types.MediaSession{
		Channels: make(map[string]srt.PubSub),
	}

	for address := 6042; address < 6142; address++ {
		if utils.IsPortFree(fmt.Sprintf("%d", address)) {
			s.Addr = ":" + strconv.Itoa(address)
			break
		}
	}

	if s.Addr == "" {
		fmt.Fprintf(os.Stderr,
			"No free port found between 6042/UDP and 6142/UDP"+
				"\nPlease ensure a port is free in this range\n")
		os.Exit(1)
	}

	s.App = "/backend"
	s.Passphrase = passphrase

	config := srt.DefaultConfig()
	config.KMPreAnnounce = 200
	config.KMRefreshRate = 10000

	s.Server = &srt.Server{
		Addr:            s.Addr,
		HandleConnect:   s.HandleConnect,
		HandlePublish:   s.HandlePublish,
		HandleSubscribe: s.HandleSubscribe,
		Config:          &config,
	}

	return s
}

func StartUpIngestSRT() *types.MediaSession {
	s := &types.MediaSession{
		Channels: make(map[string]srt.PubSub),
	}

	s.Addr = ":6001"
	s.App = "/ingest"
	//flag.StringVar(&s.addr, "addr", "", "address to listen on")
	flag.StringVar(&s.App, "app", "", "path prefix for streamid")
	flag.StringVar(&s.Token, "token", "", "token query param for streamid")
	flag.StringVar(&s.Passphrase, "passphrase", "", "passphrase for de- and enrcypting the data")
	flag.StringVar(&s.LogTopics, "logtopics", "", "topics for the log output")
	flag.StringVar(&s.Profile, "profile", "", "enable profiling (cpu, mem, allocs, heap, rate, mutex, block, thread, trace)")

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

	config := srt.DefaultConfig()

	if len(s.LogTopics) != 0 {
		config.Logger = srt.NewLogger(strings.Split(s.LogTopics, ","))
	}

	config.KMPreAnnounce = 200
	config.KMRefreshRate = 10000

	s.Server = &srt.Server{
		Addr:            s.Addr,
		HandleConnect:   s.HandleConnect,
		HandlePublish:   s.HandlePublish,
		HandleSubscribe: s.HandleSubscribe,
		Config:          &config,
	}

	fmt.Fprintf(os.Stderr, "Listening on %s\n", s.Addr)

	go func() {
		if config.Logger == nil {
			return
		}

		for m := range config.Logger.Listen() {
			fmt.Fprintf(os.Stderr, "%#08x %s (in %s:%d)\n%s \n", m.SocketId, m.Topic, m.File, m.Line, m.Message)
		}
	}()

	go func() {
		if err := s.ListenAndServe(); err != nil && !errors.Is(err, srt.ErrServerClosed) {
			fmt.Fprintf(os.Stderr, "SRT Server: %s\n", err)
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
