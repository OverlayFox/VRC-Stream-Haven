package types

import (
	"fmt"
	"github.com/OverlayFox/VRC-Stream-Haven/logger"
	srt "github.com/datarhei/gosrt"
	"net"
	"net/url"
	"os"
	"strings"
	"sync"
)

type MediaSession struct {
	// Configuration parameter taken from the Config
	Addr       string
	App        string
	Token      string
	Passphrase string
	LogTopics  string
	Profile    string

	Server *srt.Server

	// Map of publishing channels and a lock to serialize
	// access to the map.
	Channels map[string]srt.PubSub
	lock     sync.RWMutex
}

func (s *MediaSession) ListenAndServe() error {
	if len(s.App) == 0 {
		s.App = "/"
	}

	return s.Server.ListenAndServe()
}

func (s *MediaSession) Shutdown() {
	s.Server.Shutdown()
}

func (s *MediaSession) log(who, action, path, message string, client net.Addr) {
	logger.Log.Info().Msgf("%-10s %10s %s (%s) %s\n", who, action, path, client, message)
}

func (s *MediaSession) HandleConnect(req srt.ConnRequest) srt.ConnType {
	var mode srt.ConnType = srt.SUBSCRIBE
	client := req.RemoteAddr()

	channel := ""

	if req.Version() == 4 {
		mode = srt.PUBLISH
		channel = "/" + client.String()

		req.SetPassphrase(s.Passphrase)
	} else if req.Version() == 5 {
		streamId := req.StreamId()
		path := streamId

		if strings.HasPrefix(streamId, "publish:") {
			mode = srt.PUBLISH
			path = strings.TrimPrefix(streamId, "publish:")
		} else if strings.HasPrefix(streamId, "subscribe:") {
			path = strings.TrimPrefix(streamId, "subscribe:")
		}

		u, err := url.Parse(path)
		if err != nil {
			return srt.REJECT
		}

		if req.IsEncrypted() {
			if err := req.SetPassphrase(s.Passphrase); err != nil {
				s.log("CONNECT", "FORBIDDEN", u.Path, err.Error(), client)
				return srt.REJECT
			}
		}

		// Check the Token
		token := u.Query().Get("Token")
		if len(s.Token) != 0 && s.Token != token {
			s.log("CONNECT", "FORBIDDEN", u.Path, "invalid Token ("+token+")", client)
			return srt.REJECT
		}

		// Check the App patch
		if !strings.HasPrefix(u.Path, s.App) {
			s.log("CONNECT", "FORBIDDEN", u.Path, "invalid App", client)
			return srt.REJECT
		}

		println(u.Path)
		println(s.App)

		if len(strings.TrimPrefix(u.Path, s.App)) == 0 {
			s.log("CONNECT", "INVALID", u.Path, "stream name not provided", client)
			return srt.REJECT
		}

		channel = u.Path
	} else {
		return srt.REJECT
	}

	s.lock.RLock()
	pubsub := s.Channels[channel]
	s.lock.RUnlock()

	if mode == srt.PUBLISH && pubsub != nil {
		s.log("CONNECT", "CONFLICT", channel, "already publishing", client)
		return srt.REJECT
	}

	if mode == srt.SUBSCRIBE && pubsub == nil {
		s.log("CONNECT", "NOTFOUND", channel, "not publishing", client)
		return srt.REJECT
	}

	return mode
}

func (s *MediaSession) HandlePublish(conn srt.Conn) {
	channel := ""
	client := conn.RemoteAddr()
	if client == nil {
		conn.Close()
		return
	}

	if conn.Version() == 4 {
		channel = "/" + client.String()
	} else if conn.Version() == 5 {
		streamId := conn.StreamId()
		path := strings.TrimPrefix(streamId, "publish:")

		channel = path
	} else {
		s.log("PUBLISH", "INVALID", channel, "unknown connection version", client)
		conn.Close()
		return
	}

	// Look for the stream
	s.lock.Lock()
	pubsub := s.Channels[channel]
	if pubsub == nil {
		pubsub = srt.NewPubSub(srt.PubSubConfig{
			Logger: s.Server.Config.Logger,
		})
		s.Channels[channel] = pubsub
	} else {
		pubsub = nil
	}
	s.lock.Unlock()

	if pubsub == nil {
		s.log("PUBLISH", "CONFLICT", channel, "already publishing", client)
		conn.Close()
		return
	}

	s.log("PUBLISH", "START", channel, "publishing", client)

	pubsub.Publish(conn)

	s.lock.Lock()
	delete(s.Channels, channel)
	s.lock.Unlock()

	s.log("PUBLISH", "STOP", channel, "", client)

	stats := &srt.Statistics{}
	conn.Stats(stats)

	fmt.Fprintf(os.Stderr, "%+v\n", stats)

	conn.Close()
}

func (s *MediaSession) HandleSubscribe(conn srt.Conn) {
	channel := ""
	client := conn.RemoteAddr()
	if client == nil {
		conn.Close()
		return
	}

	if conn.Version() == 4 {
		channel = client.String()
	} else if conn.Version() == 5 {
		streamId := conn.StreamId()
		path := strings.TrimPrefix(streamId, "subscribe:")

		channel = path
	} else {
		s.log("SUBSCRIBE", "INVALID", channel, "unknown connection version", client)
		conn.Close()
		return
	}

	s.log("SUBSCRIBE", "START", channel, "", client)

	// Look for the stream
	s.lock.RLock()
	pubsub := s.Channels[channel]
	s.lock.RUnlock()

	if pubsub == nil {
		s.log("SUBSCRIBE", "NOTFOUND", channel, "not publishing", client)
		conn.Close()
		return
	}

	pubsub.Subscribe(conn)

	s.log("SUBSCRIBE", "STOP", channel, "", client)

	stats := &srt.Statistics{}
	conn.Stats(stats)

	fmt.Fprintf(os.Stderr, "%+v\n", stats)

	conn.Close()
}
