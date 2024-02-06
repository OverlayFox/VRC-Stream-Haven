package clients

import (
	"log"
	"time"

	"github.com/OverlayFox/VRC-Stream-Haven/servers"
	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"github.com/bluenviron/gortsplib/v4/pkg/description"
	"github.com/bluenviron/gortsplib/v4/pkg/format"
	"github.com/pion/rtp"
)

const (
	existingStream = "rtsp://x.x.x.x:8554/mystream"
	reconnectPause = 2 * time.Second
)

type client struct {
	server *servers.RtspServer
}

func (c *client) read() error {
	rc := gortsplib.Client{}

	// parse URL
	u, err := base.ParseURL(existingStream)
	if err != nil {
		return err
	}

	// connect to the server
	err = rc.Start(u.Scheme, u.Host)
	if err != nil {
		return err
	}
	defer rc.Close()

	// find available medias
	desc, _, err := rc.Describe(u)
	if err != nil {
		return err
	}

	// setup all medias
	err = rc.SetupAll(desc.BaseURL, desc.Medias)
	if err != nil {
		return err
	}

	stream := c.server.SetStreamReady(desc)
	defer c.server.SetStreamUnready()

	log.Printf("stream is ready and can be read from the server at rtsp://localhost:8554/stream\n")

	// called when a RTP packet arrives
	rc.OnPacketRTPAny(func(medi *description.Media, forma format.Format, pkt *rtp.Packet) {
		// route incoming packets to the server stream
		stream.WritePacketRTP(medi, pkt)
	})

	_, err = rc.Play(nil)
	if err != nil {
		return err
	}

	return rc.Wait()
}

func (c *client) run() {
	for {
		err := c.read()
		log.Printf("ERR: %s\n", err)

		time.Sleep(reconnectPause)
	}
}

func (c *client) initialize() {
	go c.run()
}
